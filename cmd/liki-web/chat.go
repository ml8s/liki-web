package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/records"
)

// maxAgentRunTimeout matches the bounded AG-UI product run deadline. The HTTP
// writer keeps an additional margin so a completed response can be flushed.
const maxAgentRunTimeout = 600 * time.Second

type chatWiring struct {
	Store   *chat.ChatStore
	Service *chat.Service
	Records *records.Store
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
	if config.ExpertToken == "" {
		return nil, errors.New("LIKI_EXPERT_TOKEN is required")
	}
	if config.ExpertURL == "" {
		return nil, errors.New("LIKI_EXPERT_URL is required")
	}
	timeout := time.Duration(config.ExpertTimeoutSeconds) * time.Second
	if timeout <= 0 || timeout > maxAgentRunTimeout {
		timeout = maxAgentRunTimeout
	}
	runners := make(map[string]chat.Runner, len(chat.ExpertOrder))
	runner, err := chat.NewAgentRunner(chat.AgentRunnerConfig{
		BaseURL: config.ExpertURL,
		Token:   config.ExpertToken,
		Timeout: timeout,
		Product: "liki",
	})
	if err != nil {
		return nil, fmt.Errorf("init liki aggregate agent runner: %w", err)
	}
	// Legacy session rows retain the former product-facing expert label; all
	// labels now execute on the single aggregate expert.
	for _, expert := range chat.ExpertOrder {
		runners[expert] = runner
	}
	return runners, nil
}
