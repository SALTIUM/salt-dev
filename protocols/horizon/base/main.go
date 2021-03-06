package base

import "encoding/json"

type Price struct {
	N int32 `json:"n"`
	D int32 `json:"d"`
}

type Asset struct {
	Type   string `json:"asset_type"`
	Code   string `json:"asset_code,omitempty"`
	Issuer string `json:"asset_issuer,omitempty"`
}

// Rehydratable values can be expanded in place by calling their Rehydrate
// method.  This mechanism is intended to be used for populating resource
// structs from database structs when custom logic is needed, for example if a
// resource name has been changed but the underlying database record has not.
// This interface is especially useful to facilitate field deprecation:  Add a
// new field to the response struct and implement this interface to copy the
// value from the old field to the new field.
type Rehydratable interface {
	Rehydrate() error
}

// ExtractOfferID extracts offer_id from a JSON payload
// Action needed in release: horizon-v0.25.0: remove
func ExtractOfferID(data []byte) (int64, error) {
	var temp struct {
		OfferID json.Number `json:"offer_id"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return 0, err
	}

	offerID, err := temp.OfferID.Int64()
	if err != nil {
		return 0, err
	}

	return offerID, nil
}
