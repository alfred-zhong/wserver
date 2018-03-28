package wserver

import (
	"errors"
	"fmt"
	"sync"
)

// eventConn wraps Conn with a specified event type.
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

// FindConn trys to find Conn by ID.
func (b *binder) FindConn(connID string) (*Conn, bool) {
	if connID == "" {
		return nil, false
	}

	userID, ok := b.connID2UserIDMap[connID]
	// if userID been found by connID, then find the Conn using userID
	if ok {
		if eConns, ok := b.userID2EventConnMap[userID]; ok {
			for i := range *eConns {
				if (*eConns)[i].Conn.GetID() == connID {
					return (*eConns)[i].Conn, true
				}
			}
		}

		return nil, false
	}

	// userID not found, iterate all the conns
	for _, eConns := range b.userID2EventConnMap {
		for i := range *eConns {
			if (*eConns)[i].Conn.GetID() == connID {
				return (*eConns)[i].Conn, true
			}
		}
	}

	return nil, false
}

// FilterConn searches the conns related to userID, and filtered by
// event. The userID can't be empty. The event will be ignored if it's empty.
// All the conns related to the userID will be returned if the event is empty.
func (b *binder) FilterConn(userID, event string) ([]*Conn, error) {
	if userID == "" {
		return nil, errors.New("userID can't be empty")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if eConns, ok := b.userID2EventConnMap[userID]; ok {
		ecs := make([]*Conn, 0, len(*eConns))
		for i := range *eConns {
			if event == "" || (*eConns)[i].Event == event {
				ecs = append(ecs, (*eConns)[i].Conn)
			}
		}
		return ecs, nil
	}

	return []*Conn{}, nil
}
