package zapi

import (
	"fmt"
	"testing"

	"zvelo.io/msg/go-msg"
)

func testDS(t *testing.T, ds *msg.DataSet) {
	// iterate through each valid dataset type
	for dstID := range msg.DataSetType_name {
		dst := msg.DataSetType(dstID)

		i, err := DataSetByType(ds, dst)
		if err != nil {
			t.Error("DataSetByType returned error", err)
		}

		switch dst {
		case msg.DataSetType_CATEGORIZATION:
			r, ok := i.(*msg.DataSet_Categorization)
			if !ok {
				t.Error("type of i not *msg.DataSet_Categorization")
			}

			if r != ds.Categorization {
				t.Error("t != ds.Categorization")
			}
		case msg.DataSetType_ADFRAUD:
			r, ok := i.(*msg.DataSet_AdFraud)
			if !ok {
				t.Error("type of i not *msg.DataSet_AdFraud")
			}

			if r != ds.Adfraud {
				t.Error("t != ds.Adfraud")
			}
		case msg.DataSetType_MALICIOUS:
			r, ok := i.(*msg.DataSet_Malicious)
			if !ok {
				t.Error("type of i not *msg.DataSet_Malicious")
			}

			if r != ds.Malicious {
				t.Error("t != ds.Malicious")
			}
		default:
			t.Errorf("unexpected dataset type: %s", dst)
		}
	}
}

func TestDataSetByType(t *testing.T) {
	testDS(t, &msg.DataSet{
		Categorization: &msg.DataSet_Categorization{},
		Adfraud:        &msg.DataSet_AdFraud{},
		Malicious:      &msg.DataSet_Malicious{},
	})
}

func TestNilDataSetByType(t *testing.T) {
	testDS(t, &msg.DataSet{})
}

func TestDataSetByTypeErr(t *testing.T) {
	i, err := DataSetByType(&msg.DataSet{}, msg.DataSetType(-1))

	if err == nil {
		t.Error("DataSetByType didn't return error for invalid dataset type")
	}

	if i != nil {
		t.Error("expected DataSetByType to return nil interface when err != nil ")
	}

	e, ok := err.(ErrInvalidDataSetType)
	if !ok {
		t.Error("error was not of type ErrInvalidDataSetType")
	}

	const errMsg0 = "invalid dataset type: -1"
	if e.Error() != errMsg0 || err.Error() != errMsg0 {
		t.Error("error did not have expected message")
	}

	i, err = DataSetByType(nil, msg.DataSetType_CATEGORIZATION)

	if err == nil {
		t.Error("DataSetByType didn't return error for nil dataset")
	}

	if i != nil {
		t.Error("expected DataSetByType to return nil interface when err != nil ")
	}

	if err != ErrNilDataSet {
		t.Error("unexpected error type")
	}
}

func ExampleDataSetByType() {
	ds := &msg.DataSet{
		Categorization: &msg.DataSet_Categorization{},
	}

	i, _ := DataSetByType(ds, msg.DataSetType_CATEGORIZATION)

	c := i.(*msg.DataSet_Categorization)

	fmt.Printf("c == ds.Categorization => %v\n", c == ds.Categorization)
	// Output: c == ds.Categorization => true
}
