package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"pianke-ticket/backend/internal/models"
	"pianke-ticket/backend/internal/store"

	"github.com/labstack/echo/v4"
)

const webGenerationCreditCost = 5

var creditPackages = []models.CreditPackage{
	{ID: "credits_trial", Name: "限时体验包", Credits: 10, AmountCents: 100, Currency: "CNY", Icon: "/pricing-icons/credits-100.png"},
	{ID: "credits_100", Name: "灵感入门包", Credits: 100, AmountCents: 1000, Currency: "CNY", Icon: "/pricing-icons/credits-100.png"},
	{ID: "credits_500", Name: "创作加速包", Credits: 500, AmountCents: 4900, Currency: "CNY", Icon: "/pricing-icons/credits-500.png"},
	{ID: "credits_1000", Name: "高频创作包", Credits: 1000, AmountCents: 9500, Currency: "CNY", Icon: "/pricing-icons/credits-1000.png"},
}

func (s *Server) listCreditPackages(c echo.Context) error {
	items, err := s.store.ListCreditPackages(c.Request().Context())
	if err != nil || len(items) == 0 {
		items = creditPackages
	}
	return c.JSON(http.StatusOK, models.CreditPackagesResponse{Items: items})
}

func (s *Server) purchaseCredits(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	var request models.CreditPurchaseRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "invalid purchase payload"})
	}
	packages, err := s.store.ListCreditPackages(c.Request().Context())
	if err != nil || len(packages) == 0 {
		packages = creditPackages
	}
	pkg, ok := creditPackageByIDFrom(packages, request.PackageID)
	if !ok {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "invalid credit package"})
	}
	if strings.TrimSpace(pkg.PaymentURL) != "" {
		order := models.CreditOrder{
			PackageID:   pkg.ID,
			UserID:      user.ID,
			Credits:     pkg.Credits,
			AmountCents: pkg.AmountCents,
			Currency:    pkg.Currency,
			Status:      "pending_external",
			Provider:    "card_key",
		}
		return c.JSON(http.StatusOK, models.CreditPurchaseResponse{Order: order, User: user, PaymentURL: pkg.PaymentURL})
	}
	order, err := s.store.CreateCreditOrder(c.Request().Context(), user.ID, pkg)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "purchase credits failed"})
	}
	user, err = s.store.GetUser(c.Request().Context(), user.ID)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Message: "unauthorized"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "load user failed"})
	}
	return c.JSON(http.StatusOK, models.CreditPurchaseResponse{Order: order, User: user, PaymentURL: pkg.PaymentURL})
}

func creditPackageByID(id string) (models.CreditPackage, bool) {
	return creditPackageByIDFrom(creditPackages, id)
}

func creditPackageByIDFrom(packages []models.CreditPackage, id string) (models.CreditPackage, bool) {
	id = strings.TrimSpace(id)
	for _, pkg := range packages {
		if pkg.ID == id {
			return pkg, true
		}
	}
	return models.CreditPackage{}, false
}

func (s *Server) redeemCreditCode(c echo.Context) error {
	user, ok := s.requireUser(c)
	if !ok {
		return nil
	}
	var request models.CreditRedeemRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "invalid redeem payload"})
	}
	cardNo := firstNonEmpty(strings.TrimSpace(request.CardNo), strings.TrimSpace(request.Code))
	credits, err := s.store.RedeemCreditCode(c.Request().Context(), user.ID, cardNo, request.Password)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{Message: "卡号或密码错误，或卡密已被兑换"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "兑换失败"})
	}
	user, err = s.store.GetUser(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{Message: "load user failed"})
	}
	return c.JSON(http.StatusOK, models.CreditRedeemResponse{Credits: credits, User: user})
}
