package serve

import (
	"net/http"

	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve/auth"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpdecode"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/txnbuild"
)

type accountSignHandler struct {
	Logger            *supportlog.Entry
	SigningKey        *keypair.Full
	NetworkPassphrase string
	AccountStore      account.Store
}

type accountSignRequest struct {
	Address     *keypair.FromAddress `path:"address"`
	Transaction string               `json:"transaction" form:"transaction"`
}

type accountSignResponse struct {
	PublicKey         string `json:"public_key"`
	Signature         string `json:"signature"`
	NetworkPassphrase string `json:"network_passphrase"`
}

func (h accountSignHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check that the client is authenticated in some bare minimum way.
	claims, _ := auth.FromContext(ctx)
	if claims.Address == "" && claims.PhoneNumber == "" && claims.Email == "" {
		unauthorized.Render(w)
		return
	}

	// Decode request.
	req := accountSignRequest{}
	err := httpdecode.Decode(r, &req)
	if err != nil || req.Address == nil {
		badRequest.Render(w)
		return
	}

	// Find the account that the request is for.
	acc, err := h.AccountStore.Get(req.Address.Address())
	if err == account.ErrNotFound {
		notFound.Render(w)
		return
	} else if err != nil {
		serverError.Render(w)
		return
	}

	// Verify that the authenticated client has access to the account.
	addressAuthenticated := false
	if claims.Address != "" {
		addressAuthenticated = claims.Address == acc.Address ||
			claims.Address == acc.OwnerIdentities.Address ||
			claims.Address == acc.OtherIdentities.Address
	}
	phoneNumberAuthenticated := false
	if claims.PhoneNumber != "" {
		phoneNumberAuthenticated = claims.PhoneNumber == acc.OwnerIdentities.PhoneNumber ||
			claims.PhoneNumber == acc.OtherIdentities.PhoneNumber
	}
	emailAuthenticated := false
	if claims.Email != "" {
		emailAuthenticated = claims.Email == acc.OwnerIdentities.Email ||
			claims.Email == acc.OtherIdentities.Email
	}
	if !addressAuthenticated && !phoneNumberAuthenticated && !emailAuthenticated {
		notFound.Render(w)
		return
	}

	// Decode the request transaction.
	tx, err := txnbuild.TransactionFromXDR(req.Transaction)
	if err != nil {
		badRequest.Render(w)
		return
	}
	tx.Network = h.NetworkPassphrase

	// Check that the transaction's source account and any operations it
	// contains references only to this account.
	if tx.SourceAccount.GetAccountID() != req.Address.Address() {
		badRequest.Render(w)
		return
	}
	for _, op := range tx.Operations {
		opSourceAccount := op.GetSourceAccount()
		if opSourceAccount == nil {
			continue
		}
		if op.GetSourceAccount().GetAccountID() != req.Address.Address() {
			badRequest.Render(w)
			return
		}
	}

	// Sign the transaction.
	hash, err := tx.Hash()
	if err != nil {
		h.Logger.Error(err)
		serverError.Render(w)
		return
	}
	sig, err := h.SigningKey.SignBase64(hash[:])
	if err != nil {
		h.Logger.Error(err)
		serverError.Render(w)
		return
	}
	resp := accountSignResponse{
		PublicKey:         h.SigningKey.Address(),
		Signature:         sig,
		NetworkPassphrase: h.NetworkPassphrase,
	}
	httpjson.Render(w, resp, httpjson.JSON)
}
