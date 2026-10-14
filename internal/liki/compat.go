package liki

import (
	"database/sql"
	"time"

	"context"
	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sales"
)

type Product = product.Product

const ProductChat = product.Chat

type Skill = product.Skill

const SkillLiki = product.SkillConsultation

type ProductDefinition = product.ProductDefinition

func ProductDefinitionFor(id Product) (ProductDefinition, error) { return product.DefinitionFor(id) }

type Currency = product.Currency
type CurrencyCode = product.Currency
type BillingMode = product.BillingMode

const BillingModeOneTime = product.BillingModeOneTime

type PurchaseOption = product.Offer
type Definition = product.ProductDefinition
type CommerceCatalogConfig = product.CatalogConfig
type Catalog = product.Catalog

func NewCommerceCatalog(config CommerceCatalogConfig) *Catalog { return product.NewCatalog(config) }

type Order = sales.Order
type OrderStore = sales.OrderStore
type OrderStatus = sales.OrderStatus
type Subscription = sales.Subscription
type SubscriptionStore = sales.SubscriptionStore
type SubscriptionStatus = sales.SubscriptionStatus
type SubscriptionPayment = sales.SubscriptionPayment
type SubscriptionPaymentStatus = sales.SubscriptionPaymentStatus
type Entitlement = entitlement.Entitlement
type Flow = sales.Flow
type EntitlementStore = entitlement.Store

type Fulfiller = sales.Flow

func NewFulfiller(orders *sales.OrderStore, entitlements sales.EntitlementProvisioner, productID Product, duration time.Duration) *Fulfiller {
	return sales.NewFlow(orders, entitlements, productID, duration)
}

type Service = chat.Service
type CurrencyResolver = product.CurrencyResolver
type HeaderCurrencyResolver = product.HeaderCurrencyResolver

var (
	ErrSubscriptionNotFound = sales.ErrSubscriptionNotFound
	ErrSubscriptionConflict = sales.ErrSubscriptionConflict
)

type ChatStore = chat.ChatStore

func NewSubscriptionStore(db *sql.DB) *sales.SubscriptionStore { return sales.NewSubscriptionStore(db) }
func NewOrderStore(db *sql.DB) *sales.OrderStore               { return sales.NewOrderStore(db) }
func NewOrderStoreWithCatalog(db *sql.DB, catalog *product.Catalog) *sales.OrderStore {
	return sales.NewOrderStoreWithCatalog(db, catalog)
}

const (
	OrderPaid                   = sales.OrderPaid
	SubscriptionPending         = sales.SubscriptionPending
	SubscriptionActive          = sales.SubscriptionActive
	SubscriptionPaymentRefunded = sales.SubscriptionPaymentRefunded
)

var (
	ErrSkillForbidden    = chat.ErrSkillForbidden
	ErrSessionNotFound   = chat.ErrSessionNotFound
	ErrRunnerUnavailable = chat.ErrRunnerUnavailable
)

const StatusActive = entitlement.Active

type Session = chat.Session
type Message = chat.Message
type MessageKind = chat.MessageKind
type SessionStatus = chat.SessionStatus

const (
	KindText          = chat.KindText
	KindSystemProfile = chat.KindSystemProfile
	SessionActive     = chat.SessionActive
	SessionClosed     = chat.SessionClosed
)

func NewChatStore(db *sql.DB) *chat.ChatStore { return chat.NewChatStore(db) }

func NewFlow(orders *sales.OrderStore, entitlements EntitlementProvisioner, productID Product, duration time.Duration) *Flow {
	return sales.NewFlow(orders, entitlements, productID, duration)
}

const (
	SubscriptionOnHold      = sales.SubscriptionOnHold
	SubscriptionPaymentPaid = sales.SubscriptionPaymentPaid
)

type EntitlementProvisioner = sales.EntitlementProvisioner

const CurrencySourceTrustedEdge = product.CurrencySourceTrustedEdge

func NewCurrencyResolver(source, region string) (product.CurrencyResolver, error) {
	return product.NewCurrencyResolver(source, region)
}

type AgentRunnerConfig = chat.AgentRunnerConfig
type AgentRunner = chat.AgentRunner

func NewStore(db *sql.DB) *entitlement.Store { return entitlement.NewStore(db) }

func NewSubscriptionEventProcessor(subscriptions *sales.SubscriptionStore, entitlements *entitlement.Store, providerSubscriptionProductID string, commissionObserver func(context.Context, payment.CommissionablePayment) error) *sales.SubscriptionEventProcessor {
	return sales.NewSubscriptionEventProcessor(subscriptions, entitlements, providerSubscriptionProductID, commissionObserver)
}
