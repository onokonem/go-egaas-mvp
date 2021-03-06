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

package controllers

import (
	"encoding/json"
	"github.com/EGaaS/go-egaas-mvp/packages/lib"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

func (c *Controller) GenerateNewNodeKey() (string, error) {

	priv, pub := lib.GenKeys()
	json, err := json.Marshal(map[string]string{"private_key": priv, "public_key": pub,
		"time": utils.Int64ToStr(utils.Time())})
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	log.Debug("%v", json)
	return string(json), nil
}
