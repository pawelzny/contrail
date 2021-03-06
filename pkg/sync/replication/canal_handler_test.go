package replication

import (
	"context"
	"fmt"
	"testing"

	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
	"github.com/siddontang/go-mysql/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Juniper/contrail/pkg/services"
)

func TestOnRowFailsWhenInvalidActionGiven(t *testing.T) {
	h := NewCanalHandler(&eventProcessorMock{}, &rowSinkMock{})
	err := h.OnRow(givenRowsEvent("invalid-action"))
	assert.Error(t, err)
}

func TestOnRowFailsWhenInvalidTablePrimaryKeyGiven(t *testing.T) {
	for _, action := range []string{canal.InsertAction, canal.UpdateAction, canal.DeleteAction} {
		t.Run(action, func(t *testing.T) {
			h := NewCanalHandler(&eventProcessorMock{}, &rowSinkMock{})
			e := givenRowsEvent(action)
			e.Table.PKColumns = []int{}

			err := h.OnRow(e)

			assert.Error(t, err)
		})
	}
}

func TestOnRowFailsWhenTableWithMultiColumnPrimaryKeyGiven(t *testing.T) {
	for _, action := range []string{canal.InsertAction, canal.UpdateAction, canal.DeleteAction} {
		t.Run(action, func(t *testing.T) {
			h := NewCanalHandler(&eventProcessorMock{}, &rowSinkMock{})
			e := givenRowsEvent(action)
			e.Table.PKColumns = []int{0, 1}

			err := h.OnRow(e)

			assert.Error(t, err)
		})
	}
}

func TestOnRowFailsWhenEmptyPrimaryKeyValueGiven(t *testing.T) {
	for _, action := range []string{canal.InsertAction, canal.UpdateAction, canal.DeleteAction} {
		t.Run(action, func(t *testing.T) {
			h := NewCanalHandler(&eventProcessorMock{}, &rowSinkMock{})
			e := givenRowsEvent(action)
			e.Rows = [][]interface{}{{"", 1337, 1.337}}

			err := h.OnRow(e)

			assert.Error(t, err)
		})
	}
}

func TestOnRowFailsWhenInvalidTableColumnTypeGiven(t *testing.T) {
	tests := []struct {
		action     string
		columnType int
	}{
		{canal.InsertAction, schema.TYPE_BIT},
		{canal.InsertAction, schema.TYPE_ENUM},
		{canal.InsertAction, schema.TYPE_SET},
		{canal.UpdateAction, schema.TYPE_BIT},
		{canal.UpdateAction, schema.TYPE_ENUM},
		{canal.UpdateAction, schema.TYPE_SET},
	}
	for _, test := range tests {
		t.Run(fmt.Sprint(test.action, test.columnType), func(t *testing.T) {
			h := NewCanalHandler(&eventProcessorMock{}, &rowSinkMock{})
			e := givenRowsEvent(test.action)
			e.Table.Columns = []schema.TableColumn{
				{Name: "string-property", Type: schema.TYPE_STRING},
				{Name: "property", Type: test.columnType},
			}
			e.Rows = [][]interface{}{{"foo", "property-value"}}

			err := h.OnRow(e)

			assert.Error(t, err)
		})
	}
}

func TestOnRow(t *testing.T) {
	exampleRows := [][]interface{}{{"foo", 1337, 1.337}, {"bar", 0, 0.1}, {"baz", -1337, -1.337}}
	exampleRowsData := []map[string]interface{}{
		{"string-property": "foo", "int-property": 1337, "float-property": 1.337},
		{"string-property": "bar", "int-property": 0, "float-property": 0.1},
		{"string-property": "baz", "int-property": -1337, "float-property": -1.337},
	}

	tests := []struct {
		name     string
		initMock func(oner)
		action   string
		rows     [][]interface{}
		fails    bool
	}{
		{
			name: "decode create fails",
			initMock: func(m oner) {
				m.On("DecodeRowEvent", "CREATE", mock.Anything, mock.AnythingOfType("[]string"), mock.Anything).Return(
					(*services.Event)(nil), assert.AnError,
				).Once()
			},
			action: canal.InsertAction,
			fails:  true,
		},
		{
			name: "insert 3 rows correctly",
			initMock: func(m oner) {
				m.On("DecodeRowEvent", "CREATE", "test-resource", []string{"foo"}, exampleRowsData[0]).Return(
					(*services.Event)(nil), nil,
				).Once()
				m.On("DecodeRowEvent", "CREATE", "test-resource", []string{"bar"}, exampleRowsData[1]).Return(
					(*services.Event)(nil), nil,
				).Once()
				m.On("DecodeRowEvent", "CREATE", "test-resource", []string{"baz"}, exampleRowsData[2]).Return(
					(*services.Event)(nil), nil,
				).Once()
			},
			action: canal.InsertAction,
			rows:   exampleRows,
		},
		{
			name: "decode update fails",
			initMock: func(m oner) {
				m.On("DecodeRowEvent", "UPDATE", mock.Anything, mock.AnythingOfType("[]string"), mock.Anything).Return(
					(*services.Event)(nil), assert.AnError,
				).Once()
			},
			action: canal.UpdateAction,
			fails:  true,
		},
		{
			name: "update 3 rows correctly",
			initMock: func(m oner) {
				m.On("DecodeRowEvent", "UPDATE", "test-resource", []string{"foo"}, exampleRowsData[0]).Return(
					(*services.Event)(nil), nil,
				).Once()
				m.On("DecodeRowEvent", "UPDATE", "test-resource", []string{"bar"}, exampleRowsData[1]).Return(
					(*services.Event)(nil), nil,
				).Once()
				m.On("DecodeRowEvent", "UPDATE", "test-resource", []string{"baz"}, exampleRowsData[2]).Return(
					(*services.Event)(nil), nil,
				).Once()
			},
			action: canal.UpdateAction,
			rows:   exampleRows,
		},
		{
			name: "decode delete fails",
			initMock: func(m oner) {
				m.On("DecodeRowEvent", "DELETE", mock.Anything, mock.AnythingOfType("[]string"), mock.Anything).Return(
					(*services.Event)(nil), assert.AnError,
				).Once()
			},
			action: canal.DeleteAction,
			fails:  true,
		},
		{
			name: "delete 3 rows correctly",
			initMock: func(m oner) {
				m.On("DecodeRowEvent", "DELETE", "test-resource", []string{"foo"}, (map[string]interface{})(nil)).Return(
					(*services.Event)(nil), nil,
				).Once()
				m.On("DecodeRowEvent", "DELETE", "test-resource", []string{"bar"}, (map[string]interface{})(nil)).Return(
					(*services.Event)(nil), nil,
				).Once()
				m.On("DecodeRowEvent", "DELETE", "test-resource", []string{"baz"}, (map[string]interface{})(nil)).Return(
					(*services.Event)(nil), nil,
				).Once()
			},
			action: canal.DeleteAction,
			rows:   exampleRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &rowSinkMock{}
			tt.initMock(m)
			h := NewCanalHandler(&eventProcessorMock{}, m)

			e := givenRowsEvent(tt.action)
			if tt.rows != nil {
				e.Rows = tt.rows
			}

			err := h.OnRow(e)

			if tt.fails {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			m.AssertExpectations(t)
		})
	}

}

func givenRowsEvent(action string) *canal.RowsEvent {
	return &canal.RowsEvent{
		Action: action,
		Table: &schema.Table{
			Name: "test-resource",
			Columns: []schema.TableColumn{
				{Name: "string-property", Type: schema.TYPE_STRING},
				{Name: "int-property", Type: schema.TYPE_NUMBER},
				{Name: "float-property", Type: schema.TYPE_FLOAT},
			},
			PKColumns: []int{0},
		},
		Rows: [][]interface{}{{"foo", 1337, 1.337}},
	}
}

func TestOnRotateEventIsSkipped(t *testing.T) {
	h := NewCanalHandler(&eventProcessorMock{}, nil)
	err := h.OnRotate(&replication.RotateEvent{})
	assert.NoError(t, err)
}

func TestOnDDLEventIsSkipped(t *testing.T) {
	h := NewCanalHandler(&eventProcessorMock{}, nil)
	err := h.OnDDL(mysql.Position{}, &replication.QueryEvent{})
	assert.NoError(t, err)
}

func TestOnXIDEventIsSkipped(t *testing.T) {
	h := NewCanalHandler(&eventProcessorMock{}, nil)
	err := h.OnXID(mysql.Position{})
	assert.NoError(t, err)
}

func TestOnGTIDEventIsSkipped(t *testing.T) {
	h := NewCanalHandler(&eventProcessorMock{}, nil)
	err := h.OnGTID(&mysql.MysqlGTIDSet{})
	assert.NoError(t, err)
}

func TestOnPosSyncedEventIsSkipped(t *testing.T) {
	h := NewCanalHandler(&eventProcessorMock{}, nil)
	err := h.OnPosSynced(mysql.Position{}, false)
	assert.NoError(t, err)
}

type rowSinkMock struct {
	mock.Mock
}

func (m *rowSinkMock) DecodeRowEvent(
	operation, resourceName string, pk []string, properties map[string]interface{},
) (*services.Event, error) {
	args := m.Called(operation, resourceName, pk, properties)
	return args.Get(0).(*services.Event), args.Error(1)
}

type eventProcessorMock struct {
}

func (m *eventProcessorMock) Process(ctx context.Context, e *services.Event) (*services.Event, error) {
	return nil, nil
}

func (m *eventProcessorMock) ProcessList(ctx context.Context, e *services.EventList) (*services.EventList, error) {
	return nil, nil
}
