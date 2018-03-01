package wserver

import (
	"errors"
	"fmt"
	"sync"
)

// eventConn wraps Conn with a specifed event type.
type eventConn struct {
	Event string
	Conn  *Conn
}

// binder is defined to store the relation of userID and eventConn
type binder struct {
	mu sync.RWMutex

	// map stores key: userID and value of related slice of eventConn
	userID2EventConnMap map[string]*[]eventConn

	// map stores key: connID and value: userID
	connID2UserIDMap map[string]string
}

// Bind binds userID with eConn specified by event. It fails if the
// return error is not nil.
func (b *binder) Bind(userID, event string, conn *Conn) error {
	if userID == "" {
		return errors.New("userID can't be empty")
	}

	if event == "" {
		return errors.New("event can't be empty")
	}

	if conn == nil {
		return errors.New("conn can't be nil")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// map the eConn if it isn't be put.
	if eConns, ok := b.userID2EventConnMap[userID]; ok {
		for i := range *eConns {
			if (*eConns)[i].Conn == conn {
				return nil
			}
		}

		newEConns := append(*eConns, eventConn{event, conn})
		b.userID2EventConnMap[userID] = &newEConns
	} else {
		b.userID2EventConnMap[userID] = &[]eventConn{{event, conn}}
	}
	b.connID2UserIDMap[conn.GetID()] = userID

	return nil
}

// Unbind unbind and removes Conn if it's exist.
func (b *binder) Unbind(conn *Conn) error {
	if conn == nil {
		return errors.New("conn can't be empty")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// query userID by connID
	userID, ok := b.connID2UserIDMap[conn.GetID()]
	if !ok {
		return fmt.Errorf("can't find userID by connID: %s", conn.GetID())
	}

	if eConns, ok := b.userID2EventConnMap[userID]; ok {
		for i := range *eConns {
			if (*eConns)[i].Conn == conn {
				newEConns := append((*eConns)[:i], (*eConns)[i+1:]...)
				b.userID2EventConnMap[userID] = &newEConns
				delete(b.connID2UserIDMap, conn.GetID())
				close(conn.DataChan)

				// delete the key of userID when the length of the related
				// eventConn slice is 0.
				if len(newEConns) == 0 {
					delete(b.userID2EventConnMap, userID)
				}

				return nil
			}
		}

		return fmt.Errorf("can't find the conn of ID: %s", conn.GetID())
	}

	return fmt.Errorf("can't find the eventConns by userID: %s", userID)
}

// FilterEventConn searchs the eventConns related to userID, and filtered by
// evevnt. The userID can't be empty. All the eventConns related to the userID
// will be returned if the event is empty.
func (b *binder) FilterEventConn(userID, event string) ([]eventConn, error) {
	if userID == "" {
		return nil, errors.New("userID can't be empty")
	}

	if eConns, ok := b.userID2EventConnMap[userID]; ok {
		if event == "" {
			ecs := make([]eventConn, len(*eConns))
			copy(ecs, *eConns)
			return ecs, nil
		}

		ecs := make([]eventConn, 0, len(*eConns))
		for i := range *eConns {
			if (*eConns)[i].Event == event {
				ecs = append(ecs, (*eConns)[i])
			}
		}
		return ecs, nil
	}

	return []eventConn{}, nil
}
