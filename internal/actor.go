package internalcaddyssh

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/mohammed90/caddy-ssh/internal/session"
)

type Actor struct {
	MatcherSetsRaw RawActorMatcherSet `json:"match,omitempty" caddy:"namespace=ssh.actor_matchers"`
	matcherSets    ActorMatcherSets   `json:"-"`

	ActorRaw json.RawMessage `json:"act,omitempty" caddy:"namespace=ssh.actors inline_key=action"`
	handler  session.Handler `json:"-"`

	Final bool `json:"final,omitempty"`
}

// ActorList is a list of server actors that can
// take an action on a session
type ActorList []Actor

// Provision sets up both the matchers and handlers in the routes.
func (routes ActorList) Provision(ctx caddy.Context) error {
	err := routes.ProvisionMatchers(ctx)
	if err != nil {
		return err
	}
	return routes.ProvisionHandlers(ctx)
}

// ProvisionMatchers sets up all the matchers by loading the
// matcher modules. Only call this method directly if you need
// to set up matchers and handlers separately without having
// to provision a second time; otherwise use Provision instead.
func (actors ActorList) ProvisionMatchers(ctx caddy.Context) error {
	for i := range actors {
		// matchers
		matchersIface, err := ctx.LoadModule(&actors[i], "MatcherSetsRaw")
		if err != nil {
			return fmt.Errorf("route %d: loading matcher modules: %v", i, err)
		}
		err = actors[i].matcherSets.FromInterface(matchersIface)
		if err != nil {
			return fmt.Errorf("route %d: %v", i, err)
		}
	}
	return nil
}

// ProvisionHandlers sets up all the handlers by loading the
// handler modules. Only call this method directly if you need
// to set up matchers and handlers separately without having
// to provision a second time; otherwise use Provision instead.
func (actors ActorList) ProvisionHandlers(ctx caddy.Context) error {
	for i := range actors {
		actorIface, err := ctx.LoadModule(&actors[i], "ActorRaw")
		if err != nil {
			return fmt.Errorf("route %d: loading actor modules: %v", i, err)
		}
		actors[i].handler = actorIface.(session.Handler)
	}
	return nil
}
