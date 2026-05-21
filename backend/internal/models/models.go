package models

import "time"

type ErrorResponse struct {
	Message string `json:"message"`
}

type User struct {
	ID             string    `json:"id"`
	AppID          string    `json:"appID,omitempty"`
	Email          string    `json:"email,omitempty"`
	DisplayName    string    `json:"displayName,omitempty"`
	AvatarURL      string    `json:"avatarURL,omitempty"`
	Signature      string    `json:"signature,omitempty"`
	Gender         string    `json:"gender,omitempty"`
	Credits        int       `json:"credits"`
	TotalRecharged int       `json:"totalRecharged"`
	CreatedAt      time.Time `json:"createdAt"`
}

type UserProfileUpdateRequest struct {
	DisplayName string `json:"displayName,omitempty"`
	AvatarURL   string `json:"avatarURL,omitempty"`
	Signature   string `json:"signature,omitempty"`
	Gender      string `json:"gender,omitempty"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type EmailCodeRequest struct {
	Email string `json:"email"`
	Mode  string `json:"mode,omitempty"`
	AppID string `json:"appID,omitempty"`
}

type EmailCodeResponse struct {
	ExpiresIn int    `json:"expiresIn"`
	ExpiresAt string `json:"expiresAt"`
}

type EmailCodeVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
	Mode  string `json:"mode,omitempty"`
	AppID string `json:"appID,omitempty"`
}

type EmailCodeVerifyResponse struct {
	SetupToken string `json:"setupToken"`
	ExpiresIn  int    `json:"expiresIn"`
	ExpiresAt  string `json:"expiresAt"`
}

type EmailSetupPasswordRequest struct {
	Email      string `json:"email"`
	Code       string `json:"code,omitempty"`
	SetupToken string `json:"setupToken"`
	Password   string `json:"password"`
	AppID      string `json:"appID,omitempty"`
}

type EmailPasswordLoginRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code,omitempty"`
	Password string `json:"password"`
	AppID    string `json:"appID,omitempty"`
}

type CreditPackage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Credits     int    `json:"credits"`
	AmountCents int    `json:"amountCents"`
	Currency    string `json:"currency"`
	Icon        string `json:"icon"`
	PaymentURL  string `json:"paymentURL,omitempty"`
}

type CreditPackagesResponse struct {
	Items []CreditPackage `json:"items"`
}

type CreditPurchaseRequest struct {
	PackageID string `json:"packageID"`
}

type CreditOrder struct {
	ID          string    `json:"id,omitempty"`
	UserID      string    `json:"userID"`
	PackageID   string    `json:"packageID"`
	Credits     int       `json:"credits"`
	AmountCents int       `json:"amountCents"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Provider    string    `json:"provider"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	PaidAt      time.Time `json:"paidAt,omitempty"`
}

type CreditPurchaseResponse struct {
	Order      CreditOrder `json:"order"`
	User       User        `json:"user"`
	PaymentURL string      `json:"paymentURL,omitempty"`
}

type CreditRedeemRequest struct {
	Code     string `json:"code,omitempty"`
	CardNo   string `json:"cardNo,omitempty"`
	Password string `json:"password,omitempty"`
}

type CreditRedeemResponse struct {
	Credits int  `json:"credits"`
	User    User `json:"user"`
}
