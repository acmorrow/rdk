package motionplan

import (
	"errors"

	"go.uber.org/multierr"
	pb "go.viam.com/api/component/arm/v1"

	frame "go.viam.com/rdk/referenceframe"
	spatial "go.viam.com/rdk/spatialmath"
)

// solverFrames are meant to be ephemerally created each time a frame system solution is created, and fulfills the
// Frame interface so that it can be passed to inverse kinematics.
type solverFrame struct {
	name       string
	fss        frame.FrameSystem
	movingGeom []string      // List of names of all frames that could move, used for collision detection
	frames     []frame.Frame // all frames directly between and including solveFrame and goalFrame. Order not important.
	solveFrame frame.Frame
	goalFrame  frame.Frame
	// If this is true, then goals are translated to their position in `World` before solving.
	// This is useful when e.g. moving a gripper relative to a point seen by a camera built into that gripper
	// TODO(pl): explore allowing this to be frames other than world
	worldRooted bool
	origSeed    map[string][]frame.Input // stores starting locations of all frames in fss that are NOT in `frames`
}

func newSolverFrame(
	fss frame.FrameSystem,
	solveFrameList []frame.Frame,
	goalFrameName string,
	seedMap map[string][]frame.Input,
) (*solverFrame, error) {
	var movingGeom []string
	var frames []frame.Frame
	worldRooted := false

	// get goal frame
	goalFrame := fss.Frame(goalFrameName)
	if goalFrame == nil {
		return nil, frame.NewFrameMissingError(goalFrameName)
	}
	goalFrameList, err := fss.TracebackFrame(goalFrame)
	if err != nil {
		return nil, err
	}

	// get solve frame
	if len(solveFrameList) == 0 {
		return nil, errors.New("solveFrameList was empty")
	}
	solveFrame := solveFrameList[0]

	movingGeomNames := func(frameList []frame.Frame) ([]string, error) {
		// Find first moving frame
		var moveF frame.Frame
		for i := len(frameList) - 1; i >= 0; i-- {
			if len(frameList[i].DoF()) != 0 {
				moveF = frameList[i]
				break
			}
		}
		if moveF == nil {
			return []string{}, nil
		}
		movingFs, err := fss.FrameSystemSubset(moveF)
		if err != nil {
			return nil, err
		}
		return movingFs.FrameNames(), nil
	}

	// find pivot frame between goal and solve frames
	pivotFrame, err := findPivotFrame(solveFrameList, goalFrameList)
	if err != nil {
		return nil, err
	}
	if pivotFrame.Name() == frame.World {
		frames = uniqInPlaceSlice(append(solveFrameList, goalFrameList...))
		movingGeom, err = movingGeomNames(solveFrameList)
		if err != nil {
			return nil, err
		}
		movingGeom2, err := movingGeomNames(goalFrameList)
		if err != nil {
			return nil, err
		}
		movingGeom = append(movingGeom, movingGeom2...)
	} else {
		dof := 0
		var solveMovingList []frame.Frame
		var goalMovingList []frame.Frame

		// Get minimal set of frames from solve frame to goal frame
		for _, frame := range solveFrameList {
			if frame == pivotFrame {
				break
			}
			dof += len(frame.DoF())
			frames = append(frames, frame)
			solveMovingList = append(solveMovingList, frame)
		}
		for _, frame := range goalFrameList {
			if frame == pivotFrame {
				break
			}
			dof += len(frame.DoF())
			frames = append(frames, frame)
			goalMovingList = append(goalMovingList, frame)
		}

		// If shortest path has 0 dof (e.g. a camera attached to a gripper), translate goal to world frame
		if dof == 0 {
			worldRooted = true
			frames = solveFrameList
			movingGeom, err = movingGeomNames(solveFrameList)
			if err != nil {
				return nil, err
			}
		} else {
			// Get all child nodes of pivot node
			movingGeom, err = movingGeomNames(solveMovingList)
			if err != nil {
				return nil, err
			}
			movingGeom2, err := movingGeomNames(goalMovingList)
			if err != nil {
				return nil, err
			}
			movingGeom = append(movingGeom, movingGeom2...)
		}
	}

	origSeed := map[string][]frame.Input{}
	// deep copy of seed map
	for k, v := range seedMap {
		origSeed[k] = v
	}
	for _, frame := range frames {
		delete(origSeed, frame.Name())
	}

	return &solverFrame{
		name:        solveFrame.Name() + "_" + goalFrame.Name(),
		fss:         fss,
		movingGeom:  movingGeom,
		frames:      frames,
		solveFrame:  solveFrame,
		goalFrame:   goalFrame,
		worldRooted: worldRooted,
		origSeed:    origSeed,
	}, nil
}

// Name returns the name of the solver referenceframe.
func (sf *solverFrame) Name() string {
	return sf.name
}

// Transform returns the pose between the two frames of this solver for a given set of inputs.
func (sf *solverFrame) Transform(inputs []frame.Input) (spatial.Pose, error) {
	if len(inputs) != len(sf.DoF()) {
		return nil, frame.NewIncorrectInputLengthError(len(inputs), len(sf.DoF()))
	}
	pf := frame.NewPoseInFrame(sf.solveFrame.Name(), spatial.NewZeroPose())
	solveName := sf.goalFrame.Name()
	if sf.worldRooted {
		solveName = frame.World
	}
	tf, err := sf.fss.Transform(sf.sliceToMap(inputs), pf, solveName)
	if err != nil {
		return nil, err
	}
	return tf.(*frame.PoseInFrame).Pose(), nil
}

// InputFromProtobuf converts pb.JointPosition to inputs.
func (sf *solverFrame) InputFromProtobuf(jp *pb.JointPositions) []frame.Input {
	inputs := make([]frame.Input, 0, len(jp.Values))
	posIdx := 0
	for _, transform := range sf.frames {
		dof := len(transform.DoF()) + posIdx
		jPos := jp.Values[posIdx:dof]
		posIdx = dof

		inputs = append(inputs, transform.InputFromProtobuf(&pb.JointPositions{Values: jPos})...)
	}
	return inputs
}

// ProtobufFromInput converts inputs to pb.JointPosition.
func (sf *solverFrame) ProtobufFromInput(input []frame.Input) *pb.JointPositions {
	jPos := &pb.JointPositions{}
	posIdx := 0
	for _, transform := range sf.frames {
		dof := len(transform.DoF()) + posIdx
		jPos.Values = append(jPos.Values, transform.ProtobufFromInput(input[posIdx:dof]).Values...)
		posIdx = dof
	}
	return jPos
}

// Geometry takes a solverFrame and a list of joint angles in radians and computes the 3D space occupied by each of the
// geometries in the solverFrame in the reference frame of the World frame.
func (sf *solverFrame) Geometries(inputs []frame.Input) (*frame.GeometriesInFrame, error) {
	if len(inputs) != len(sf.DoF()) {
		return nil, frame.NewIncorrectInputLengthError(len(inputs), len(sf.DoF()))
	}
	var errAll error
	inputMap := sf.sliceToMap(inputs)
	sfGeometries := make(map[string]spatial.Geometry)
	for _, fName := range sf.movingGeom {
		f := sf.fss.Frame(fName)
		if f == nil {
			return nil, frame.NewFrameMissingError(fName)
		}
		inputs, err := frame.GetFrameInputs(f, inputMap)
		if err != nil {
			return nil, err
		}
		gf, err := f.Geometries(inputs)
		if gf == nil {
			// only propagate errors that result in nil geometry
			multierr.AppendInto(&errAll, err)
			continue
		}
		var tf frame.Transformable
		tf, err = sf.fss.Transform(inputMap, gf, frame.World)
		if err != nil {
			return nil, err
		}
		for name, geometry := range tf.(*frame.GeometriesInFrame).Geometries() {
			sfGeometries[name] = geometry
		}
	}
	return frame.NewGeometriesInFrame(frame.World, sfGeometries), errAll
}

// DoF returns the summed DoF of all frames between the two solver frames.
func (sf *solverFrame) DoF() []frame.Limit {
	var limits []frame.Limit
	for _, frame := range sf.frames {
		limits = append(limits, frame.DoF()...)
	}
	return limits
}

// mapToSlice will flatten a map of inputs into a slice suitable for input to inverse kinematics, by concatenating
// the inputs together in the order of the frames in sf.frames.
func (sf *solverFrame) mapToSlice(inputMap map[string][]frame.Input) ([]frame.Input, error) {
	var inputs []frame.Input
	for _, f := range sf.frames {
		input, err := frame.GetFrameInputs(f, inputMap)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input...)
	}
	return inputs, nil
}

func (sf *solverFrame) sliceToMap(inputSlice []frame.Input) map[string][]frame.Input {
	inputs := map[string][]frame.Input{}
	for k, v := range sf.origSeed {
		inputs[k] = v
	}
	i := 0
	for _, frame := range sf.frames {
		fLen := i + len(frame.DoF())
		inputs[frame.Name()] = inputSlice[i:fLen]
		i = fLen
	}
	return inputs
}

func (sf *solverFrame) MarshalJSON() ([]byte, error) {
	return nil, errors.New("cannot serialize solverFrame")
}

func (sf *solverFrame) AlmostEquals(otherFrame frame.Frame) bool {
	return false
}

// uniqInPlaceSlice will deduplicate the values in a slice using in-place replacement on the slice. This is faster than
// a solution using append().
// This function does not remove anything from the input slice, but it does rearrange the elements.
func uniqInPlaceSlice(s []frame.Frame) []frame.Frame {
	seen := make(map[frame.Frame]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

// findPivotFrame finds the first common frame in two ordered lists of frames.
func findPivotFrame(frameList1, frameList2 []frame.Frame) (frame.Frame, error) {
	// find shorter list
	shortList := frameList1
	longList := frameList2
	if len(frameList1) > len(frameList2) {
		shortList = frameList2
		longList = frameList1
	}

	// cache names seen in shorter list
	nameSet := make(map[string]struct{}, len(shortList))
	for _, frame := range shortList {
		nameSet[frame.Name()] = struct{}{}
	}

	// look for already seen names in longer list
	for _, frame := range longList {
		if _, ok := nameSet[frame.Name()]; ok {
			return frame, nil
		}
	}
	return nil, errors.New("no path from solve frame to goal frame")
}