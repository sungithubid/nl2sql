// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package nl2sql

import (
	"context"

	v1 "nl2sql/api/nl2sql/v1"
)

type INl2sqlV1 interface {
	TrainDDL(ctx context.Context, req *v1.TrainDDLReq) (res *v1.TrainDDLRes, err error)
	TrainDoc(ctx context.Context, req *v1.TrainDocReq) (res *v1.TrainDocRes, err error)
	TrainSQL(ctx context.Context, req *v1.TrainSQLReq) (res *v1.TrainSQLRes, err error)
	Ask(ctx context.Context, req *v1.AskReq) (res *v1.AskRes, err error)
	RemoveTrainingData(ctx context.Context, req *v1.RemoveTrainingDataReq) (res *v1.RemoveTrainingDataRes, err error)
	ListTrainingData(ctx context.Context, req *v1.ListTrainingDataReq) (res *v1.ListTrainingDataRes, err error)
}
