// Copyright 2016 The go-daylight Authors
// This file is part of the go-daylight library.
//
// The go-daylight library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-daylight library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-daylight library. If not, see <http://www.gnu.org/licenses/>.

package parser

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

// не использовать для комментов
func (p *Parser) selectiveLoggingAndUpd(fields []string, values_ []interface{}, table string, whereFields, whereValues []string, generalRollback bool) (string, error) {
	var (
		tableId  string
		isCustom bool
		err      error
	)

	if generalRollback && p.BlockData == nil {
		return ``, fmt.Errorf(`It is impossible to write to DB when Block is undefined`)
	}

	isBytea := getBytea(table)
	if isCustom, err = p.IsCustomTable(table); err != nil {
		return ``, err
	}

	for i, v := range values_ {
		if len(fields) > i && isBytea[fields[i]] {
			var vlen int
			switch v.(type) {
			case []byte:
				vlen = len(v.([]byte))
			case string:
				if vbyte, err := hex.DecodeString(v.(string)); err == nil {
					values_[i] = vbyte
					vlen = len(vbyte)
				} else {
					vlen = len(v.(string))
				}
			}
			if isCustom && vlen > 32 {
				return ``, fmt.Errorf(`hash value cannot be larger than 32 bytes`)
			}
		}
	}

	values := utils.InterfaceSliceToStr(values_)

	addSqlFields := p.AllPkeys[table]
	if len(addSqlFields) > 0 {
		addSqlFields += `,`
	}
	log.Debug("addSqlFields %s", addSqlFields)
	for i, field := range fields {
		/*if p.AllPkeys[table] == field {
			continue
		}*/
		field = strings.TrimSpace(field)
		fields[i] = field
		if field[:1] == "+" || field[:1] == "-" {
			addSqlFields += field[1:len(field)] + ","
		} else if strings.HasPrefix(field, `timestamp `) {
			addSqlFields += field[len(`timestamp `):] + `,`
		} else {
			addSqlFields += field + ","
		}
	}
	log.Debug("addSqlFields %s", addSqlFields)

	addSqlWhere := ""
	if whereFields != nil && whereValues != nil {
		for i := 0; i < len(whereFields); i++ {
			addSqlWhere += whereFields[i] + "= '" + whereValues[i] + "' AND "
		}
	}
	if len(addSqlWhere) > 0 {
		addSqlWhere = " WHERE " + addSqlWhere[0:len(addSqlWhere)-5]
	}
	// если есть, что логировать
	logData, err := p.OneRow(`SELECT ` + addSqlFields + ` rb_id FROM "` + table + `" ` + addSqlWhere).String()
	if err != nil {
		return tableId, err
	}
	log.Debug(`SELECT ` + addSqlFields + ` rb_id FROM "` + table + `" ` + addSqlWhere)
	if whereFields != nil && len(logData) > 0 {
		jsonMap := make(map[string]string)
		for k, v := range logData {
			if k == p.AllPkeys[table] {
				continue
			}
			if (isBytea[k] || utils.InSliceString(k, []string{"hash", "tx_hash", "public_key_0", "public_key_1", "public_key_2", "node_public_key"})) && v != "" {
				jsonMap[k] = string(utils.BinToHex([]byte(v)))
			} else {
				jsonMap[k] = v
			}
			if k == "rb_id" {
				k = "prev_rb_id"
			}
			if k[:1] == "+" || k[:1] == "-" {
				addSqlFields += k[1:len(k)] + ","
			} else if strings.HasPrefix(k, `timestamp `) {
				addSqlFields += k[len(`timestamp `):] + `,`
			} else {
				addSqlFields += k + ","
			}
		}
		jsonData, _ := json.Marshal(jsonMap)
		if err != nil {
			return tableId, err
		}
		rbId, err := p.ExecSqlGetLastInsertId("INSERT INTO rollback ( data, block_id ) VALUES ( ?, ? )", "rollback", string(jsonData), p.BlockData.BlockId)
		if err != nil {
			return tableId, err
		}
		log.Debug("string(jsonData) %s / rbId %d", string(jsonData), rbId)
		addSqlUpdate := ""
		for i := 0; i < len(fields); i++ {
			// utils.InSliceString(fields[i], []string{"hash", "tx_hash", "public_key", "public_key_0", "public_key_1", "public_key_2", "node_public_key"}
			if isBytea[fields[i]] && len(values[i]) != 0 {
				addSqlUpdate += fields[i] + `=decode('` + hex.EncodeToString([]byte(values[i])) + `','HEX'),`
			} else if fields[i][:1] == "+" {
				addSqlUpdate += fields[i][1:len(fields[i])] + `=` + fields[i][1:len(fields[i])] + `+` + values[i] + `,`
			} else if fields[i][:1] == "-" {
				addSqlUpdate += fields[i][1:len(fields[i])] + `=` + fields[i][1:len(fields[i])] + `-` + values[i] + `,`
			} else if values[i] == `NULL` {
				addSqlUpdate += fields[i] + `= NULL,`
			} else if strings.HasPrefix(fields[i], `timestamp `) {
				addSqlUpdate += fields[i][len(`timestamp `):] + `= to_timestamp('` + values[i] + `'),`
			} else if strings.HasPrefix(values[i], `timestamp `) {
				addSqlUpdate += fields[i] + `= timestamp '` + values[i][len(`timestamp `):] + `',`
			} else {
				addSqlUpdate += fields[i] + `='` + strings.Replace(values[i], `'`, `''`, -1) + `',`
			}
		}
		err = p.ExecSql(`UPDATE "`+table+`" SET `+addSqlUpdate+` rb_id = ? `+addSqlWhere, rbId)
		log.Debug(`UPDATE "` + table + `" SET ` + addSqlUpdate + ` rb_id = ? ` + addSqlWhere)
		//log.Debug("logId", logId)
		if err != nil {
			return tableId, err
		}
		tableId = logData[p.AllPkeys[table]]
	} else {
		addSqlIns0 := ""
		addSqlIns1 := ""
		for i := 0; i < len(fields); i++ {
			if fields[i][:1] == "+" || fields[i][:1] == "-" {
				addSqlIns0 += fields[i][1:len(fields[i])] + `,`
			} else if strings.HasPrefix(fields[i], `timestamp `) {
				addSqlIns0 += fields[i][len(`timestamp `):] + `,`
			} else {
				addSqlIns0 += fields[i] + `,`
			}
			// || utils.InSliceString(fields[i], []string{"hash", "tx_hash", "public_key", "public_key_0", "public_key_1", "public_key_2", "node_public_key"}))
			if isBytea[fields[i]] && len(values[i]) != 0 {
				addSqlIns1 += `decode('` + hex.EncodeToString([]byte(values[i])) + `','HEX'),`
			} else if values[i] == `NULL` {
				addSqlIns1 += `NULL,`
			} else if strings.HasPrefix(fields[i], `timestamp `) {
				addSqlIns1 += `to_timestamp('` + values[i] + `'),`
			} else if strings.HasPrefix(values[i], `timestamp `) {
				addSqlIns1 += `timestamp '` + values[i][len(`timestamp `):] + `',`
			} else {
				addSqlIns1 += `'` + strings.Replace(values[i], `'`, `''`, -1) + `',`
			}
		}
		if whereFields != nil && whereValues != nil {
			for i := 0; i < len(whereFields); i++ {
				addSqlIns0 += `` + whereFields[i] + `,`
				addSqlIns1 += `'` + whereValues[i] + `',`
			}
		}
		addSqlIns0 = addSqlIns0[0 : len(addSqlIns0)-1]
		addSqlIns1 = addSqlIns1[0 : len(addSqlIns1)-1]
		//		fmt.Println(`Sel Log`, "INSERT INTO "+table+" ("+addSqlIns0+") VALUES ("+addSqlIns1+")")
		tableId, err = p.ExecSqlGetLastInsertId(`INSERT INTO "`+table+`" (`+addSqlIns0+`) VALUES (`+addSqlIns1+`)`, table)
		if err != nil {
			return tableId, err
		}
	}
	if generalRollback {
		err = p.ExecSql("INSERT INTO rollback_tx ( block_id, tx_hash, table_name, table_id ) VALUES (?, [hex], ?, ?)", p.BlockData.BlockId, p.TxHash, table, tableId)
		if err != nil {
			return tableId, err
		}
	}
	return tableId, nil
}
