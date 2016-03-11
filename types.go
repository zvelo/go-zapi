package zapi

import "time"

type token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

type DataSetType string

const (
	DataSetTypeCategorization DataSetType = "categorization"
	DataSetTypeAdFraud        DataSetType = "adfraud"
)

type QueryURLRequests struct {
	URLs           []string      `json:"urls,omitempty"`
	DataSets       []DataSetType `json:"datasets,omitempty"`
	CallbackURL    string        `json:"callback,omitempty"`
	PartialResults bool          `json:"partial-results,omitempty"`
}

type Status struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type QueryReply struct {
	Status     *Status  `json:"status,omitempty"`
	RequestIDs [][]byte `json:"request_id,omitempty"`
	StatusURL  string   `json:"status_url,omitempty"`
}

type QueryResult struct {
	RequestID     []byte        `json:"request_id,omitempty"`
	TrackingID    *string       `json:"tracking_id,omitempty"`
	URL           *ParsedURL    `json:"url,omitempty"`
	Status        *Status       `json:"status,omitempty"`
	RequestStatus *Status       `json:"request_status,omitempty"`
	RequestDS     []DataSetType `json:"request_ds,omitempty"`
	ResponseDS    *DataSet      `json:"response_ds,omitempty"`
}

type ParsedURL struct {
	OriginalURL *string `json:"original_url,omitempty"`
	Error       string  `json:"error,omitempty"`
	URL         URL     `json:"url,omitempty"`
}

type UserInfo struct {
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

type URL struct {
	Scheme   *string   `json:"scheme,omitempty"`
	Opaque   *string   `json:"opaque,omitempty"`
	User     *UserInfo `json:"user,omitempty"`
	Host     *string   `json:"host,omitempty"`
	Path     *string   `json:"path,omitempty"`
	RawQuery *string   `json:"raw_query,omitempty"`
	Fragment *string   `json:"fragment,omitempty"`
}

type DataSet struct {
	Categorization *DataSetCategorization `json:"c,omitempty"`
	AdFraud        *DataSetAdFraud        `json:"af,omitempty"`
}

type DataSetCategorization struct {
	ID []int32 `json:"id,omitempty"`
}

type DataSetAdFraud struct {
	Fraud     bool   `json:"fraud,omitempty"`
	Signature string `json:"signature,omitempty"`
}
