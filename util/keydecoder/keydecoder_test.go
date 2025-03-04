// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package keydecoder_test

import (
	"testing"

	"github.com/pingcap/parser/model"
	"github.com/pingcap/tidb/infoschema"
	"github.com/pingcap/tidb/sessionctx/stmtctx"
	"github.com/pingcap/tidb/table"
	"github.com/pingcap/tidb/table/tables"
	"github.com/pingcap/tidb/testkit"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/codec"
	. "github.com/pingcap/tidb/util/keydecoder"
	"github.com/stretchr/testify/assert"
)

func TestDecodeKey(t *testing.T) {
	table.MockTableFromMeta = tables.MockTableFromMeta
	tableInfo1 := &model.TableInfo{
		ID:   1,
		Name: model.NewCIStr("table1"),
		Indices: []*model.IndexInfo{
			{ID: 1, Name: model.NewCIStr("index1"), State: model.StatePublic},
		},
	}
	tableInfo2 := &model.TableInfo{ID: 2, Name: model.NewCIStr("table2")}
	tableInfo3 := &model.TableInfo{
		ID:   3,
		Name: model.NewCIStr("table3"),
		Columns: []*model.ColumnInfo{
			{ID: 10, Name: model.NewCIStr("col"), State: model.StatePublic},
		},
		Indices: []*model.IndexInfo{
			{ID: 4, Name: model.NewCIStr("index4"), State: model.StatePublic},
		},
		Partition: &model.PartitionInfo{
			Type:   model.PartitionTypeRange,
			Expr:   "`col`",
			Enable: true,
			Definitions: []model.PartitionDefinition{
				{ID: 5, Name: model.NewCIStr("p0"), LessThan: []string{"10"}},
				{ID: 6, Name: model.NewCIStr("p1"), LessThan: []string{"MAXVALUE"}},
			},
		},
	}

	stubTableInfos := []*model.TableInfo{tableInfo1, tableInfo2, tableInfo3}
	stubInfoschema := infoschema.MockInfoSchema(stubTableInfos)

	decodedKey, err := DecodeKey([]byte{
		't',
		// table id = 1
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		'_',
		'r',
		// int handle, value = 1
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}, stubInfoschema)
	assert.Nil(t, err)
	assert.Equal(t, decodedKey.DbID, int64(0))
	assert.Equal(t, decodedKey.DbName, "test")
	assert.Equal(t, decodedKey.TableID, int64(1))
	assert.Equal(t, decodedKey.TableName, "table1")
	assert.Equal(t, decodedKey.PartitionID, int64(0))
	assert.Equal(t, decodedKey.PartitionName, "")
	assert.Equal(t, decodedKey.HandleType, IntHandle)
	assert.Equal(t, decodedKey.IsPartitionHandle, false)
	assert.Equal(t, decodedKey.HandleValue, "1")
	// These are default values, ie. will be omitted when got marshaled into json
	assert.Equal(t, decodedKey.IndexID, int64(0))
	assert.Equal(t, decodedKey.IndexName, "")
	assert.Nil(t, decodedKey.IndexValues)

	ch := testkit.MustNewCommonHandle(t, 100, "abc")
	encodedCommonKey := ch.Encoded()
	key := []byte{
		't',
		// table id = 2
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
		'_',
		'r',
	}
	key = append(key, encodedCommonKey...)

	decodedKey, err = DecodeKey(key, stubInfoschema)
	assert.Nil(t, err)
	assert.Equal(t, decodedKey.DbID, int64(0))
	assert.Equal(t, decodedKey.DbName, "test")
	assert.Equal(t, decodedKey.TableID, int64(2))
	assert.Equal(t, decodedKey.TableName, "table2")
	assert.Equal(t, decodedKey.PartitionID, int64(0))
	assert.Equal(t, decodedKey.PartitionName, "")
	assert.Equal(t, decodedKey.HandleType, CommonHandle)
	assert.Equal(t, decodedKey.IsPartitionHandle, false)
	assert.Equal(t, decodedKey.HandleValue, "{100, abc}")
	// These are default values, ie. will be omitted when got marshaled into json
	assert.Equal(t, decodedKey.IndexID, int64(0))
	assert.Equal(t, decodedKey.IndexName, "")
	assert.Nil(t, decodedKey.IndexValues)

	values := types.MakeDatums("abc", 1)
	sc := &stmtctx.StatementContext{}
	encodedValue, err := codec.EncodeKey(sc, nil, values...)
	assert.Nil(t, err)
	key = []byte{
		't',
		// table id = 1
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		'_',
		'i',
		// index id = 1
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	key = append(key, encodedValue...)

	decodedKey, err = DecodeKey(key, stubInfoschema)
	assert.Nil(t, err)
	assert.Equal(t, decodedKey.DbID, int64(0))
	assert.Equal(t, decodedKey.DbName, "test")
	assert.Equal(t, decodedKey.TableID, int64(1))
	assert.Equal(t, decodedKey.TableName, "table1")
	assert.Equal(t, decodedKey.PartitionID, int64(0))
	assert.Equal(t, decodedKey.PartitionName, "")
	assert.Equal(t, decodedKey.IndexID, int64(1))
	assert.Equal(t, decodedKey.IndexName, "index1")
	assert.Equal(t, decodedKey.IndexValues, []string{"abc", "1"})
	// These are default values, ie. will be omitted when got marshaled into json
	assert.Equal(t, decodedKey.HandleType, HandleType(""))
	assert.Equal(t, decodedKey.HandleValue, "")
	assert.Equal(t, decodedKey.IsPartitionHandle, false)

	// Row key in a partitioned table.
	key = []byte("t\x80\x00\x00\x00\x00\x00\x00\x05_r\x80\x00\x00\x00\x00\x00\x00\x0a")
	decodedKey, err = DecodeKey(key, stubInfoschema)
	assert.Nil(t, err)
	assert.Equal(t, decodedKey.DbID, int64(0))
	assert.Equal(t, decodedKey.DbName, "test")
	assert.Equal(t, decodedKey.TableID, int64(3))
	assert.Equal(t, decodedKey.TableName, "table3")
	assert.Equal(t, decodedKey.PartitionID, int64(5))
	assert.Equal(t, decodedKey.PartitionName, "p0")
	assert.Equal(t, decodedKey.HandleType, IntHandle)
	assert.Equal(t, decodedKey.HandleValue, "10")
	// These are default values, ie. will be omitted when got marshaled into json
	assert.Equal(t, decodedKey.IndexID, int64(0))
	assert.Equal(t, decodedKey.IndexName, "")
	assert.Nil(t, decodedKey.IndexValues)
	assert.Equal(t, decodedKey.IsPartitionHandle, false)

	// Index key in a partitioned table.
	values = types.MakeDatums("abcde", 2)
	encodedValue, err = codec.EncodeKey(sc, nil, values...)
	assert.Nil(t, err)
	key = []byte("t\x80\x00\x00\x00\x00\x00\x00\x06_i\x80\x00\x00\x00\x00\x00\x00\x04")
	key = append(key, encodedValue...)

	decodedKey, err = DecodeKey(key, stubInfoschema)
	assert.Nil(t, err)
	assert.Equal(t, decodedKey.DbID, int64(0))
	assert.Equal(t, decodedKey.DbName, "test")
	assert.Equal(t, decodedKey.TableID, int64(3))
	assert.Equal(t, decodedKey.TableName, "table3")
	assert.Equal(t, decodedKey.PartitionID, int64(6))
	assert.Equal(t, decodedKey.PartitionName, "p1")
	assert.Equal(t, decodedKey.IndexID, int64(4))
	assert.Equal(t, decodedKey.IndexName, "index4")
	assert.Equal(t, decodedKey.IndexValues, []string{"abcde", "2"})
	// These are default values, ie. will be omitted when got marshaled into json
	assert.Equal(t, decodedKey.HandleType, HandleType(""))
	assert.Equal(t, decodedKey.HandleValue, "")
	assert.Equal(t, decodedKey.IsPartitionHandle, false)

	// Totally invalid key
	key = []byte("this-is-a-totally-invalidkey")
	decodedKey, err = DecodeKey(key, stubInfoschema)
	assert.NotNil(t, err)
	// "partly" invalid key, i.e. neither record nor index
	key = []byte{
		't',
		// table id = 1
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	key = append(key, []byte("rest-part-is-invalid")...)
	decodedKey, err = DecodeKey(key, stubInfoschema)
	assert.NotNil(t, err)

	// Table cannot be found in infoschema
	// This is possible when the schema have changed since when the key is get.
	decodedKey, err = DecodeKey([]byte{
		't',
		// table id = 3
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
		'_',
		'r',
		// int handle, value = 1
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}, stubInfoschema)
	// We should get as much information as we can
	assert.Nil(t, err)
	assert.Equal(t, decodedKey.TableID, int64(4))
	assert.Equal(t, decodedKey.HandleType, IntHandle)
	assert.Equal(t, decodedKey.HandleValue, "1")

	// Rest information are all default value, ie. omitted when got marshaled into json
	assert.Equal(t, decodedKey.DbID, int64(0))
	assert.Equal(t, decodedKey.DbName, "")
	assert.Equal(t, decodedKey.TableName, "")
	assert.Equal(t, decodedKey.PartitionID, int64(0))
	assert.Equal(t, decodedKey.PartitionName, "")
	assert.Equal(t, decodedKey.IndexID, int64(0))
	assert.Equal(t, decodedKey.IndexName, "")
	assert.Equal(t, decodedKey.IsPartitionHandle, false)
	assert.Nil(t, decodedKey.IndexValues)
}
