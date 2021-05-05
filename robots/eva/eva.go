package eva

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.viam.com/robotcore/api"
	"go.viam.com/robotcore/kinematics"
	pb "go.viam.com/robotcore/proto/api/v1"
	"go.viam.com/robotcore/utils"

	"github.com/edaniels/golog"
)

func init() {
	api.RegisterArm("eva", func(ctx context.Context, r api.Robot, config api.ComponentConfig, logger golog.Logger) (api.Arm, error) {
		return NewEva(ctx, config.Host, config.Attributes, logger)
	})
}

type evaData struct {
	// map[estop:false]
	Global map[string]interface{}

	// map[d0:false d1:false d2:false d3:false ee_a0:0.034 ee_a1:0.035 ee_d0:false ee_d1:false]
	GlobalInputs map[string]interface{} `json:"global.inputs"`

	//map[d0:false d1:false d2:false d3:false ee_d0:false ee_d1:false]
	GlobalOutputs map[string]interface{} `json:"global.outputs"`

	//scheduler : map[enabled:false]
	Scheduler map[string]interface{}

	//[0.0008628905634395778 0 0.0002876301878131926 0 -0.00038350690738298 0.0005752603756263852]
	ServosPosition []float64 `json:"servos.telemetry.position"`

	//[53.369998931884766 43.75 43.869998931884766 43.869998931884766 51 48.619998931884766]
	ServosTemperature []float64 `json:"servos.telemetry.temperature"`

	//[0 0 0 0 0 0]
	ServosVelocity []float64 `json:"servos.telemetry.velocity"`

	//map[loop_count:1 loop_target:1 run_mode:not_running state:ready toolpath_hash:4d8 toolpath_name:Uploaded]
	Control map[string]interface{}
}

type eva struct {
	host         string
	version      string
	token        string
	sessionToken string

	moveLock sync.Mutex
	logger   golog.Logger
}

func (e *eva) CurrentJointPositions(ctx context.Context) (*pb.JointPositions, error) {
	data, err := e.DataSnapshot(ctx)
	if err != nil {
		return &pb.JointPositions{}, err
	}
	return api.JointPositionsFromRadians(data.ServosPosition), nil
}

func (e *eva) CurrentPosition(ctx context.Context) (*pb.ArmPosition, error) {
	return nil, fmt.Errorf("eva low level doesn't support kinematics")
}

func (e *eva) MoveToPosition(ctx context.Context, pos *pb.ArmPosition) error {
	return fmt.Errorf("eva low level doesn't support kinematics")
}

func (e *eva) MoveToJointPositions(ctx context.Context, newPositions *pb.JointPositions) error {
	radians := api.JointPositionsToRadians(newPositions)

	err := e.doMoveJoints(ctx, radians)
	if err == nil {
		return nil
	}

	if !strings.Contains(err.Error(), "Reset hard errors first") {
		return err
	}

	err2 := e.resetErrors(ctx)
	if err2 != nil {
		return fmt.Errorf("move failure, and couldn't reset errors %s %s", err, err2)
	}

	return e.doMoveJoints(ctx, radians)
}

func (e *eva) doMoveJoints(ctx context.Context, joints []float64) error {
	e.moveLock.Lock()
	defer e.moveLock.Unlock()

	err := e.apiLock(ctx)
	if err != nil {
		return err
	}
	defer e.apiUnlock(ctx)

	return e.apiControlGoTo(ctx, joints)
}

func (e *eva) JointMoveDelta(ctx context.Context, joint int, amount float64) error {
	return fmt.Errorf("not done yet")
}

func (e *eva) Close() error {
	return nil
}

func (e *eva) apiRequest(ctx context.Context, method string, path string, payload interface{}, auth bool, out interface{}) error {
	return e.apiRequestRetry(ctx, method, path, payload, auth, out, true)
}

func (e *eva) apiRequestRetry(ctx context.Context, method string, path string, payload interface{}, auth bool, out interface{}, retry bool) error {
	fullPath := fmt.Sprintf("http://%s/api/%s/%s", e.host, e.version, path)

	var reqReader io.Reader = nil
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		reqReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, fullPath, reqReader)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	if auth {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", e.sessionToken))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 401 {
		// need to login

		if !retry {
			return fmt.Errorf("got 401 from eva after trying to login")
		}

		type Temp struct {
			Token string
		}
		t := Temp{}
		err = e.apiRequestRetry(ctx, "POST", "auth", map[string]string{"token": e.token}, false, &t, false)
		if err != nil {
			return err
		}

		e.sessionToken = t.Token
		return e.apiRequestRetry(ctx, method, path, payload, auth, out, false)
	}

	if res.StatusCode != 200 {
		more := ""
		if res.Body != nil {
			more2, e2 := ioutil.ReadAll(res.Body)
			if e2 == nil {
				more = string(more2)
			}
		}

		return fmt.Errorf("got unexpected response code: %d for %s %s", res.StatusCode, fullPath, more)
	}

	if out == nil {
		return nil
	}

	if !strings.HasPrefix(res.Header["Content-Type"][0], "application/json") {
		return fmt.Errorf("expected json response from eva, got: %v", res.Header["Content-Type"])
	}

	decoder := json.NewDecoder(res.Body)

	return decoder.Decode(out)
}

func (e *eva) apiName(ctx context.Context) (string, error) {
	type Temp struct {
		Name string
	}
	t := Temp{}

	err := e.apiRequest(ctx, "GET", "name", nil, false, &t)

	if err != nil {
		return "", err
	}

	return t.Name, nil
}

func (e *eva) resetErrors(ctx context.Context) error {
	e.moveLock.Lock()
	defer e.moveLock.Unlock()

	err := e.apiLock(ctx)
	if err != nil {
		return err
	}
	defer e.apiUnlock(ctx)

	err = e.apiRequest(ctx, "POST", "controls/reset_errors", nil, true, nil)
	if err != nil {
		return err
	}
	utils.SelectContextOrWait(ctx, 100*time.Millisecond)
	return ctx.Err()
}

func (e *eva) DataSnapshot(ctx context.Context) (evaData, error) {
	type Temp struct {
		Snapshot evaData
	}
	res := Temp{}

	err := e.apiRequest(ctx, "GET", "data/snapshot", nil, true, &res)
	return res.Snapshot, err
}

func (e *eva) apiControlGoTo(ctx context.Context, joints []float64) error {
	body := map[string]interface{}{"joints": joints, "mode": "teach"} // TODO(erh): change to automatic
	err := e.apiRequest(ctx, "POST", "controls/go_to", &body, true, nil)
	if err != nil {
		return err
	}

	return nil
}

func (e *eva) apiLock(ctx context.Context) error {
	return e.apiRequest(ctx, "POST", "controls/lock", nil, true, nil)
}

func (e *eva) apiUnlock(ctx context.Context) {
	err := e.apiRequest(ctx, "DELETE", "controls/lock", nil, true, nil)
	if err != nil {
		e.logger.Debugf("eva unlock failed: %s", err)
	}
}

func NewEva(ctx context.Context, host string, attrs api.AttributeMap, logger golog.Logger) (api.Arm, error) {
	e := &eva{
		host:    host,
		version: "v1",
		token:   attrs.GetString("token"),
		logger:  logger,
	}

	name, err := e.apiName(ctx)
	if err != nil {
		return nil, err
	}

	e.logger.Debugf("connected to eva: %v", name)

	return kinematics.NewArmJSONFile(e, attrs.GetString("modelJSON"), 4, logger)
}
