package wserver

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

var bindingLock sync.RWMutex

// token -> 员工 ID 绑定
var employeeIDBinding = make(map[string]int64)

// 员工 ID -> 数据 channel 绑定
var chanBinding = make(map[int64]*[]dataChan)

type dataChan struct {
	Ch    chan []byte
	Event string
}

// 绑定数据通道
func bind(token, event string, ch chan []byte) error {
	if token == "" {
		return errors.New("token 不能为空")
	}

	if event == "" {
		return errors.New("event 不能为空")
	}

	if ch == nil {
		return errors.New("data channel 不能为空")
	}

	employeeID, err := findEmployeeIDByToken(token)
	if err != nil {
		return err
	}

	bindingLock.Lock()
	defer bindingLock.Unlock()

	// token -> 员工 ID 绑定
	employeeIDBinding[token] = employeeID

	// 员工 ID -> 数据 channel 绑定
	if dataChanSlice, ok := chanBinding[employeeID]; ok {
		for i := range *dataChanSlice {
			if (*dataChanSlice)[i].Ch == ch {
				return nil
			}
		}

		newDataChanSlice := append(*dataChanSlice, dataChan{ch, event})
		chanBinding[employeeID] = &newDataChanSlice
	} else {
		chanBinding[employeeID] = &[]dataChan{dataChan{ch, event}}
	}
	return nil
}

// 解绑数据通道
func unbind(token string, ch chan []byte) error {
	if token == "" {
		return errors.New("token 不能为空")
	}

	if ch == nil {
		return errors.New("data channel 不能为空")
	}

	bindingLock.Lock()
	defer bindingLock.Unlock()

	// 根据 token 查询相应的员工 ID
	employeeID, ok := employeeIDBinding[token]
	if !ok {
		return fmt.Errorf("解绑失败，token: %v 找不到对应员工 ID", token)
	}

	// 员工 ID -> 数据 channel 解绑
	if dataChanSlice, ok := chanBinding[employeeID]; ok {
		for i := range *dataChanSlice {
			if (*dataChanSlice)[i].Ch == ch {
				newDataChanSlice := append((*dataChanSlice)[:i], (*dataChanSlice)[i+1:]...)
				chanBinding[employeeID] = &newDataChanSlice
				close(ch)

				// 当员工 ID 对应的数据通道数量为 0 时，删除 token -> 员工 ID 绑定
				// 和员工 ID -> 数据 channel 绑定
				if len(newDataChanSlice) == 0 {
					delete(employeeIDBinding, token)
					delete(chanBinding, employeeID)
				}

				return nil
			}
		}

		return fmt.Errorf("解绑失败，Channel: %v 找不到对应", ch)
	}

	return fmt.Errorf("解绑失败，员工 ID: %d 找不到对应数据 Channel", employeeID)
}

// 发送信息
func sendMessage(employeeID int64, event string, message []byte) error {
	if event == "" {
		return errors.New("事件类型不能为空")
	}

	if message == nil {
		return errors.New("发送数据不能为空")
	}

	bindingLock.RLock()

	// 查询可用的通道发送数据
	cnt := 0
	if dataChanSlice, ok := chanBinding[employeeID]; ok {
		for i := range *dataChanSlice {
			dc := (*dataChanSlice)[i]
			if dc.Event == event {
				select {
				case dc.Ch <- message:
					cnt++
				default:
					newDataChanSlice := append((*dataChanSlice)[:i], (*dataChanSlice)[i+1:]...)
					dataChanSlice = &newDataChanSlice
					close(dc.Ch)
				}
			}
		}
	}
	bindingLock.RUnlock()

	log.Printf("向员工: %d 推送了事件类型: %s 数据 %d 条", employeeID, event, cnt)
	return nil
}

// 根据 token 查询对应的员工 ID
func findEmployeeIDByToken(token string) (int64, error) {
	// FIXME:
	return 0, nil
}
