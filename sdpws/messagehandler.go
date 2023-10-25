package sdpws

import (
	"context"

	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
)

type GatewayMessageHandler interface {
	NewItem(context.Context, *sdp.Item)
	NewEdge(context.Context, *sdp.Edge)
	Status(context.Context, *sdp.GatewayRequestStatus)
	Error(context.Context, string)
	QueryError(context.Context, *sdp.QueryError)
	DeleteItem(context.Context, *sdp.Reference)
	DeleteEdge(context.Context, *sdp.Edge)
	UpdateItem(context.Context, *sdp.Item)
	SnapshotStoreResult(context.Context, *sdp.SnapshotStoreResult)
	SnapshotLoadResult(context.Context, *sdp.SnapshotLoadResult)
	BookmarkStoreResult(context.Context, *sdp.BookmarkStoreResult)
	BookmarkLoadResult(context.Context, *sdp.BookmarkLoadResult)
	QueryStatus(context.Context, *sdp.QueryStatus)
}

type LoggingGatewayMessageHandler struct {
	Level log.Level
}

// assert that LoggingGatewayMessageHandler implements GatewayMessageHandler
var _ GatewayMessageHandler = (*LoggingGatewayMessageHandler)(nil)

func (l *LoggingGatewayMessageHandler) NewItem(ctx context.Context, item *sdp.Item) {
	log.WithContext(ctx).WithField("item", item).Log(l.Level, "received new item")
}

func (l *LoggingGatewayMessageHandler) NewEdge(ctx context.Context, edge *sdp.Edge) {
	log.WithContext(ctx).WithField("edge", edge).Log(l.Level, "received new edge")
}

func (l *LoggingGatewayMessageHandler) Status(ctx context.Context, status *sdp.GatewayRequestStatus) {
	log.WithContext(ctx).WithField("status", status).Log(l.Level, "received status")
}

func (l *LoggingGatewayMessageHandler) Error(ctx context.Context, errorMessage string) {
	log.WithContext(ctx).WithField("errorMessage", errorMessage).Log(l.Level, "received error")
}

func (l *LoggingGatewayMessageHandler) QueryError(ctx context.Context, queryError *sdp.QueryError) {
	log.WithContext(ctx).WithField("queryError", queryError).Log(l.Level, "received query error")
}

func (l *LoggingGatewayMessageHandler) DeleteItem(ctx context.Context, reference *sdp.Reference) {
	log.WithContext(ctx).WithField("reference", reference).Log(l.Level, "received delete item")
}

func (l *LoggingGatewayMessageHandler) DeleteEdge(ctx context.Context, edge *sdp.Edge) {
	log.WithContext(ctx).WithField("edge", edge).Log(l.Level, "received delete edge")
}

func (l *LoggingGatewayMessageHandler) UpdateItem(ctx context.Context, item *sdp.Item) {
	log.WithContext(ctx).WithField("item", item).Log(l.Level, "received updated item")
}

func (l *LoggingGatewayMessageHandler) SnapshotStoreResult(ctx context.Context, result *sdp.SnapshotStoreResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received snapshot store result")
}

func (l *LoggingGatewayMessageHandler) SnapshotLoadResult(ctx context.Context, result *sdp.SnapshotLoadResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received snapshot load result")
}

func (l *LoggingGatewayMessageHandler) BookmarkStoreResult(ctx context.Context, result *sdp.BookmarkStoreResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received bookmark store result")
}

func (l *LoggingGatewayMessageHandler) BookmarkLoadResult(ctx context.Context, result *sdp.BookmarkLoadResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received bookmark load result")
}

func (l *LoggingGatewayMessageHandler) QueryStatus(ctx context.Context, status *sdp.QueryStatus) {
	log.WithContext(ctx).WithField("status", status).Log(l.Level, "received query status")
}
