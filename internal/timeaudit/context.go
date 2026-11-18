package timeaudit

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrProtectedEntry = errors.New("manager cannot edit protected time entries")

var ErrUnauthenticated = errors.New("authenticated time audit actor is required")

type Actor struct {
	ID      string
	Type    string
	Source  string
	Comment string
}
type actorKey struct{}
type httpKey struct{}
type systemKey struct{}
type managerKey struct{}
type managerScope struct{ ActorID, OwnerID string }

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}
func FromContext(ctx context.Context) Actor {
	actor, _ := ctx.Value(actorKey{}).(Actor)
	if actor.Source == "" {
		actor.Source = "user_time_entries"
	}
	return actor
}

// RequestActor only accepts the action types already authorized by current HTTP routes.
func RequestActor(ctx context.Context, actor Actor) (context.Context, error) {
	if strings.TrimSpace(actor.ID) == "" || (actor.Type != "employee" && actor.Type != "admin") {
		return nil, ErrUnauthenticated
	}
	return WithActor(context.WithValue(ctx, httpKey{}, true), actor), nil
}

// WithSystem is explicit and cannot turn an HTTP action into a system action.
func WithSystem(ctx context.Context, source string) context.Context {
	ctx = context.WithValue(ctx, systemKey{}, true)
	return WithActor(ctx, Actor{Type: "system", Source: source})
}

// WithManager is called only after permission and department scope validation.
func WithManager(ctx context.Context, id, owner, comment string) context.Context {
	ctx = context.WithValue(ctx, managerKey{}, managerScope{ActorID: id, OwnerID: owner})
	return WithActor(ctx, Actor{ID: id, Type: "manager", Source: "calendar", Comment: comment})
}
func Validate(ctx context.Context) error {
	actor := FromContext(ctx)
	if actor.Type == "system" {
		internal, _ := ctx.Value(systemKey{}).(bool)
		request, _ := ctx.Value(httpKey{}).(bool)
		if !internal || request || actor.ID != "" {
			return ErrUnauthenticated
		}
		return nil
	}
	if strings.TrimSpace(actor.ID) == "" {
		return ErrUnauthenticated
	}
	switch actor.Type {
	case "employee", "admin":
		return nil
	case "manager":
		verified, _ := ctx.Value(managerKey{}).(managerScope)
		if verified.ActorID != actor.ID || verified.OwnerID == "" || verified.OwnerID == actor.ID || strings.TrimSpace(actor.Comment) == "" || utf8.RuneCountInString(actor.Comment) > 4000 {
			return ErrUnauthenticated
		}
		return nil
	default:
		return ErrUnauthenticated
	}
}
func WithSource(ctx context.Context, source string) context.Context {
	actor := FromContext(ctx)
	actor.Source = source
	return WithActor(ctx, actor)
}
func WithUser(ctx context.Context, actorID, ownerID, source string) context.Context {
	actorType := "admin"
	if actorID == ownerID {
		actorType = "employee"
	}
	return WithActor(ctx, Actor{ID: actorID, Type: actorType, Source: source, Comment: FromContext(ctx).Comment})
}

func ManagerOwns(ctx context.Context, owner string) bool {
	scope, _ := ctx.Value(managerKey{}).(managerScope)
	return scope.OwnerID == owner && scope.ActorID == FromContext(ctx).ID && scope.ActorID != owner
}
