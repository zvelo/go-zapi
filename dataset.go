package zapi

import (
	"fmt"
	"reflect"
	"strings"

	"zvelo.io/msg/go-msg"
)

// ErrInvalidDataSetType indicates an invalid int was used as a DataSetType enum
// value
type ErrInvalidDataSetType msg.DataSetType

func (e ErrInvalidDataSetType) Error() string {
	return fmt.Sprintf("invalid dataset type: %d", int32(e))
}

// Various errors
var (
	ErrNilDataSet   = fmt.Errorf("dataset was nil")
	ErrInvalidField = fmt.Errorf("dataset type does not exist in dataset definition")
)

// DataSetByType returns one of the field values of a msg.DataSet based on a
// given dsType. It determines which value to return by doing a case insensitive
// comparison of msg.DataSetType.String() and the field name of msg.DataSet. It
// returns an interface{} that can be type asserted into the appropriate message
// type.
func DataSetByType(ds *msg.DataSet, dsType msg.DataSetType) (interface{}, error) {
	name, ok := msg.DataSetType_name[int32(dsType)]
	if !ok {
		return nil, ErrInvalidDataSetType(dsType)
	}

	if ds == nil {
		return nil, ErrNilDataSet
	}

	v := reflect.ValueOf(*ds)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if strings.ToLower(t.Field(i).Name) != strings.ToLower(name) {
			continue
		}

		return v.Field(i).Interface(), nil
	}

	// NOTE: if this is reached, it indicates a problem where a valid
	// DataSetType was provided, but DataSet has no cooresponding
	// (case-insensitive) field name
	return nil, ErrInvalidField
}
