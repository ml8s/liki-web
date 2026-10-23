package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/records"
	"github.com/ml8s/liki-web/internal/runcontrol"
)

// maxAgentRunTimeout matches the bounded AG-UI product run deadline. The HTTP
// writer keeps an additional margin so a completed response can be flushed.
const maxAgentRunTimeout = 600 * time.Second

type chatWiring struct {
	Store              *chat.ChatStore
	Service            *chat.Service
	Records            *records.Store
	RunRecords         *runcontrol.Store
	RunCallbackHandler http.HandlerFunc
}

func newChatWiring(
	ctx context.Context,
	config Config,
	db *sql.DB,
	entitlements *entitlement.Store,
	profiles *birthprofile.Store,
) (*chatWiring, error) {
	system := &chatWiring{Store: chat.NewChatStore(db)}
	if err := system.Store.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("init chat schema: %w", err)
	}
	runControl := runcontrol.NewStore(db)
	system.RunRecords = runControl
	if err := runControl.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("init run registry schema: %w", err)
	}
	if config.RunCallbackSecret != "" {
		handler, err := runcontrol.NewCallbackHandler(runcontrol.CallbackConfig{
			Secret:  config.RunCallbackSecret,
			Store:   runControl,
			History: system.Store,
		})
		if err != nil {
			return nil, fmt.Errorf("init run callback receiver: %w", err)
		}
		system.RunCallbackHandler = handler
	}
	system.Records = records.NewStore(db)
	if err := system.Records.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("init records schema: %w", err)
	}
	system.Service = chat.NewService(system.Store, entitlements, profiles)
	runners, err := newAgentRunners(config)
	if err != nil {
		return nil, err
	}
	system.Service.SetAgentRunners(runners)
	return system, nil
}

func newAgentRunners(config Config) (map[string]chat.Runner, error) {
	if config.AgentInternalToken == "" {
		return nil, errors.New("LIKI_AGENTS_INTERNAL_TOKEN is required")
	}
	timeout := time.Duration(config.AgentTimeoutSeconds) * time.Second
	if timeout <= 0 || timeout > maxAgentRunTimeout {
		timeout = maxAgentRunTimeout
	}
	runners := make(map[string]chat.Runner, len(chat.ExpertOrder))
	for _, expert := range chat.ExpertOrder {
		baseURL := config.AgentURLs[expert]
		if baseURL == "" {
			return nil, fmt.Errorf("missing agent endpoint: expert=%s env=LIKI_AGENTS_%s_URL", expert, strings.ToUpper(expert))
		}
		runner, err := chat.NewAgentRunner(chat.AgentRunnerConfig{
			BaseURL: baseURL,
			Token:   config.AgentInternalToken,
			Timeout: timeout,
			Product: "liki",
		})
		if err != nil {
			return nil, fmt.Errorf("init liki agent runner %s: %w", expert, err)
		}
		runners[expert] = runner
	}
	return runners, nil
}
