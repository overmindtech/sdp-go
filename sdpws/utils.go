package sdpws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
)

func (c *Client) SendQuery(ctx context.Context, q *sdp.Query) error {
	if c.Closed() {
		return errors.New("client closed")
	}

	log.WithContext(ctx).WithField("query", q).Trace("writing query to websocket")
	err := c.send(ctx, &sdp.GatewayRequest{
		RequestType: &sdp.GatewayRequest_Query{
			Query: q,
		},
		MinStatusInterval: durationpb.New(time.Second),
	})
	if err != nil {
		// c.send already aborts
		// c.abort(ctx, err)
		return err
	}
	return nil
}

func (c *Client) Query(ctx context.Context, q *sdp.Query) ([]*sdp.Item, error) {
	if c.Closed() {
		return nil, errors.New("client closed")
	}

	r := c.createRequestChan(uuid.UUID(q.UUID))

	err := c.SendQuery(ctx, q)
	if err != nil {
		// c.SendQuery already aborts
		// c.abort(ctx, err)
		return nil, err
	}

	items := make([]*sdp.Item, 0)

readLoop:
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled: %w", ctx.Err())
		case resp, more := <-r:
			if !more {
				break readLoop
			}
			switch resp.ResponseType.(type) {
			case *sdp.GatewayResponse_NewItem:
				item := resp.GetNewItem()
				log.WithContext(ctx).WithField("query", q).WithField("item", item).Debug("received item")
				items = append(items, item)
			case *sdp.GatewayResponse_QueryError:
				qe := resp.GetQueryError()
				log.WithContext(ctx).WithField("query", q).WithField("queryerror", qe).Trace("received query error")
				// ignore query errors
				// switch qe.ErrorType {
				// case sdp.QueryError_OTHER:
				// 	return nil, fmt.Errorf("query error: %v", qe.ErrorString)
				// case sdp.QueryError_TIMEOUT:
				// 	return nil, fmt.Errorf("query timeout: %v", qe.ErrorString)
				// case sdp.QueryError_NOSCOPE:
				// 	return nil, fmt.Errorf("query to wrong scope: %v", qe.ErrorString)
				// case sdp.QueryError_NOTFOUND:
				// 	continue readLoop
				// }
				continue readLoop
			case *sdp.GatewayResponse_QueryStatus:
				qs := resp.GetQueryStatus()
				log.WithContext(ctx).WithField("query", q).WithField("querystatus", qs).Trace("received query status")
				switch qs.Status {
				case sdp.QueryStatus_FINISHED:
					break readLoop
				case sdp.QueryStatus_CANCELLED:
					return nil, errors.New("query cancelled")
				case sdp.QueryStatus_ERRORED:
					// if we already received items, we can ignore the error
					if len(items) == 0 {
						err = errors.New("query errored")
						// query errors should not abort the connection
						// c.abort(ctx, err)
						return nil, err
					}
					break readLoop
				}
			default:
				log.WithContext(ctx).WithField("response", resp).WithField("responseType", fmt.Sprintf("%T", resp.ResponseType)).Warn("unexpected response")
			}
		}
	}

	c.finishRequestChan(uuid.UUID(q.UUID))
	return items, nil
}

// TODO: CancelQuery
// TODO: UndoQuery
// TODO: Expand
// TODO: UndoExpand
// TODO: StoreSnapshot
// TODO: LoadSnapshot

func (c *Client) SendStoreBookmark(ctx context.Context, b *sdp.StoreBookmark) error {
	if c.Closed() {
		return errors.New("client closed")
	}

	log.WithContext(ctx).WithField("bookmark", b).Trace("storing bookmark via websocket")
	err := c.send(ctx, &sdp.GatewayRequest{
		RequestType: &sdp.GatewayRequest_StoreBookmark{
			StoreBookmark: b,
		},
	})
	return err
}

func (c *Client) StoreBookmark(ctx context.Context, name, description string, isSystem bool) (uuid.UUID, error) {
	if c.Closed() {
		return uuid.UUID{}, errors.New("client closed")
	}

	u := uuid.New()
	b := &sdp.StoreBookmark{
		Name:        name,
		Description: description,
		MsgID:       u[:],
		IsSystem:    true,
	}
	r := c.createRequestChan(u)

	err := c.SendStoreBookmark(ctx, b)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("error sending store bookmark request: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return uuid.UUID{}, fmt.Errorf("context canceled: %w", ctx.Err())
		case resp, more := <-r:
			if !more {
				return uuid.UUID{}, errors.New("request channel closed")
			}
			switch resp.ResponseType.(type) {
			case *sdp.GatewayResponse_BookmarkStoreResult:
				bsr := resp.GetBookmarkStoreResult()
				log.WithContext(ctx).WithField("bookmark", b).WithField("bookmarkstoreresult", bsr).Trace("received bookmark store result")
				if bsr.Success {
					return uuid.UUID(bsr.BookmarkID), nil
				}
				return uuid.UUID{}, fmt.Errorf("bookmark store failed: %v", bsr.ErrorMessage)
			default:
				log.WithContext(ctx).WithField("response", resp).WithField("responseType", fmt.Sprintf("%T", resp.ResponseType)).Warn("unexpected response")
				return uuid.UUID{}, errors.New("unexpected response")
			}
		}
	}
}

// TODO: LoadBookmark
