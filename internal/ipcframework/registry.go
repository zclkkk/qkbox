package ipcframework

import (
	"context"
	"encoding/json"

	"github.com/zclkkk/qkbox/shared/api"
)

type MethodHandler func(context.Context, json.RawMessage) (interface{}, *api.StructuredError)

type SubscriptionHandler func(context.Context, json.RawMessage) (<-chan api.RuntimeEvent, interface{}, *api.StructuredError)

type Registry struct {
	source        string
	methods       map[string]MethodHandler
	subscriptions map[string]SubscriptionHandler
}

func NewRegistry(source string) *Registry {
	return &Registry{
		source:        source,
		methods:       map[string]MethodHandler{},
		subscriptions: map[string]SubscriptionHandler{},
	}
}

func RegisterMethod[Req any, Reply any](registry *Registry, method string, handler func(context.Context, Req) (Reply, *api.StructuredError)) {
	registry.methods[method] = func(ctx context.Context, payload json.RawMessage) (interface{}, *api.StructuredError) {
		var req Req
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), registry.source, true)
		}
		return handler(ctx, req)
	}
}

func RegisterSubscription[Req any](registry *Registry, method string, ack interface{}, handler func(context.Context, Req) (<-chan api.RuntimeEvent, *api.StructuredError)) {
	registry.subscriptions[method] = func(ctx context.Context, payload json.RawMessage) (<-chan api.RuntimeEvent, interface{}, *api.StructuredError) {
		var req Req
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, nil, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), registry.source, true)
		}
		events, structured := handler(ctx, req)
		if structured != nil {
			return nil, nil, structured
		}
		return events, ack, nil
	}
}

func (r *Registry) Method(name string) (MethodHandler, bool) {
	handler, ok := r.methods[name]
	return handler, ok
}

func (r *Registry) Subscription(name string) (SubscriptionHandler, bool) {
	handler, ok := r.subscriptions[name]
	return handler, ok
}
