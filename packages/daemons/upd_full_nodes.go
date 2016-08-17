package daemons

import (
	"fmt"
	"github.com/DayLightProject/go-daylight/packages/parser"
	"github.com/DayLightProject/go-daylight/packages/utils"
	"github.com/DayLightProject/go-daylight/packages/consts"
)

func UpdFullNodes(chBreaker chan bool, chAnswer chan string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("daemon Recovered", r)
			panic(r)
		}
	}()

	const GoroutineName = "UpdFullNodes"
	d := new(daemon)
	d.DCDB = DbConnect(chBreaker, chAnswer, GoroutineName)
	if d.DCDB == nil {
		return
	}
	d.goRoutineName = GoroutineName
	d.chAnswer = chAnswer
	d.chBreaker = chBreaker
	if utils.Mobile() {
		d.sleepTime = 3600
	} else {
		d.sleepTime = 60
	}
	if !d.CheckInstall(chBreaker, chAnswer, GoroutineName) {
		return
	}
	d.DCDB = DbConnect(chBreaker, chAnswer, GoroutineName)
	if d.DCDB == nil {
		return
	}

BEGIN:
	for {
		logger.Info(GoroutineName)
		MonitorDaemonCh <- []string{GoroutineName, utils.Int64ToStr(utils.Time())}

		// проверим, не нужно ли нам выйти из цикла
		if CheckDaemonsRestart(chBreaker, chAnswer, GoroutineName) {
			break BEGIN
		}

		err, restart := d.dbLock()
		if restart {
			break BEGIN
		}
		if err != nil {
			if d.dPrintSleep(err, d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}

		blockId, err := d.GetBlockId()
		if err != nil {
			if d.unlockPrintSleep(utils.ErrInfo(err), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}
		if blockId == 0 {
			if d.unlockPrintSleep(utils.ErrInfo("blockId == 0"), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}

		myCBID, myWalletId, err := d.GetMyCBIDAndWalletId();
		logger.Debug("%v", myWalletId)
		if err != nil {
			d.dbUnlock()
			logger.Error("%v", err)
			if d.dSleep(d.sleepTime) {
				break BEGIN
			}
			continue
		}

		// Есть ли мы в списке тех, кто может генерить блоки
		full_node_id, err:= d.FindInFullNodes(myCBID, myWalletId)
		if err != nil {
			d.dbUnlock()
			logger.Error("%v", err)
			if d.dSleep(d.sleepTime) {
				break BEGIN
			}
			continue
		}
		logger.Debug("full_node_id = %d", full_node_id)
		if full_node_id == 0 {
			d.dbUnlock()
			logger.Debug("full_node_id == 0")
			d.sleepTime = 10
			if d.dSleep(d.sleepTime) {
				break BEGIN
			}
			continue
		}

		curTime := utils.Time()

		// проверим, прошло ли время с момента последнего обновления
		upd_full_nodes, err := d.Single("SELECT time FROM upd_full_nodes").Int64()
		if err != nil {
			if d.unlockPrintSleep(utils.ErrInfo(err), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}
		if curTime-upd_full_nodes <= consts.UPD_FULL_NODES_PERIOD {
			if d.unlockPrintSleep(utils.ErrInfo("curTime-adminTime <= consts.UPD_FULL_NODES_PERIO"), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}

		forSign := fmt.Sprintf("%v,%v,%v,%v", utils.TypeInt("UpdFullNodes"), curTime, myWalletId, myCBID)
		binSign, err := d.GetBinSign(forSign)
		if err != nil {
			if d.unlockPrintSleep(utils.ErrInfo(err), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}
		data := utils.DecToBin(utils.TypeInt("UpdFullNodes"), 1)
		data = append(data, utils.DecToBin(curTime, 4)...)
		data = append(data, utils.EncodeLengthPlusData(myWalletId)...)
		data = append(data, utils.EncodeLengthPlusData(myCBID)...)
		data = append(data, utils.EncodeLengthPlusData([]byte(binSign))...)

		err = d.InsertReplaceTxInQueue(data)
		if err != nil {
			if d.unlockPrintSleep(utils.ErrInfo(err), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}

		p := new(parser.Parser)
		p.DCDB = d.DCDB
		err = p.TxParser(utils.HexToBin(utils.Md5(data)), data, true)
		if err != nil {
			if d.unlockPrintSleep(err, d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}

		d.dbUnlock()

		if d.dSleep(d.sleepTime) {
			break BEGIN
		}
	}
	logger.Debug("break BEGIN %v", GoroutineName)
}