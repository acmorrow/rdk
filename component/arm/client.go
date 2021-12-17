// Package arm contains a gRPC based arm client.
package arm

import (
	"context"

	"github.com/edaniels/golog"
	"go.viam.com/utils/rpc"

	"go.viam.com/core/grpc"
	commonpb "go.viam.com/core/proto/api/common/v1"
	pb "go.viam.com/core/proto/api/component/v1"
	"go.viam.com/core/referenceframe"
)

// serviceClient is a client satisfies the arm.proto contract.
type serviceClient struct {
	conn   rpc.ClientConn
	client pb.ArmServiceClient
	logger golog.Logger
}

// newServiceClient constructs a new serviceClient that is served at the given address.
func newServiceClient(ctx context.Context, address string, logger golog.Logger, opts ...rpc.DialOption) (*serviceClient, error) {
	conn, err := grpc.Dial(ctx, address, logger, opts...)
	if err != nil {
		return nil, err
	}
	sc := newSvcClientFromConn(conn, logger)
	return sc, nil
}

// newSvcClientFromConn constructs a new serviceClient using the passed in connection.
func newSvcClientFromConn(conn rpc.ClientConn, logger golog.Logger) *serviceClient {
	client := pb.NewArmServiceClient(conn)
	sc := &serviceClient{
		conn:   conn,
		client: client,
		logger: logger,
	}
	return sc
}

// Close cleanly closes the underlying connections
func (sc *serviceClient) Close() error {
	return sc.conn.Close()
}

// client is an arm client
type client struct {
	*serviceClient
	name string
}

// NewClient constructs a new client that is served at the given address.
func NewClient(ctx context.Context, name string, address string, logger golog.Logger, opts ...rpc.DialOption) (Arm, error) {
	sc, err := newServiceClient(ctx, address, logger, opts...)
	if err != nil {
		return nil, err
	}
	return clientFromSvcClient(sc, name), nil
}

// NewClientFromConn constructs a new Client from connection passed in.
func NewClientFromConn(conn rpc.ClientConn, name string, logger golog.Logger) Arm {
	sc := newSvcClientFromConn(conn, logger)
	return clientFromSvcClient(sc, name)
}

func clientFromSvcClient(sc *serviceClient, name string) Arm {
	return &client{sc, name}
}

func (c *client) CurrentPosition(ctx context.Context) (*commonpb.Pose, error) {
	resp, err := c.client.CurrentPosition(ctx, &pb.ArmServiceCurrentPositionRequest{
		Name: c.name,
	})
	if err != nil {
		return nil, err
	}
	return resp.Position, nil
}

func (c *client) MoveToPosition(ctx context.Context, pos *commonpb.Pose) error {
	_, err := c.client.MoveToPosition(ctx, &pb.ArmServiceMoveToPositionRequest{
		Name: c.name,
		To:   pos,
	})
	return err
}

func (c *client) MoveToJointPositions(ctx context.Context, pos *pb.ArmJointPositions) error {
	_, err := c.client.MoveToJointPositions(ctx, &pb.ArmServiceMoveToJointPositionsRequest{
		Name: c.name,
		To:   pos,
	})
	return err
}

func (c *client) CurrentJointPositions(ctx context.Context) (*pb.ArmJointPositions, error) {
	resp, err := c.client.CurrentJointPositions(ctx, &pb.ArmServiceCurrentJointPositionsRequest{
		Name: c.name,
	})
	if err != nil {
		return nil, err
	}
	return resp.Positions, nil
}

func (c *client) JointMoveDelta(ctx context.Context, joint int, amountDegs float64) error {
	_, err := c.client.JointMoveDelta(ctx, &pb.ArmServiceJointMoveDeltaRequest{
		Name:       c.name,
		Joint:      int32(joint),
		AmountDegs: amountDegs,
	})
	return err
}

func (c *client) ModelFrame() *referenceframe.Model {
	// TODO(erh): this feels wrong
	return nil
}

func (c *client) CurrentInputs(ctx context.Context) ([]referenceframe.Input, error) {
	res, err := c.CurrentJointPositions(ctx)
	if err != nil {
		return nil, err
	}
	return referenceframe.JointPosToInputs(res), nil
}

func (c *client) GoToInputs(ctx context.Context, goal []referenceframe.Input) error {
	return c.MoveToJointPositions(ctx, referenceframe.InputsToJointPos(goal))
}

// Close cleanly closes the underlying connections
func (c *client) Close() error {
	return c.serviceClient.Close()
}
