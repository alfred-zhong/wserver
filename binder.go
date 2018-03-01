package wserver

type binder struct {
	token2userIDMap     map[string]string
	userID2EventConnMap map[string]*[]eventConn

	CalcUserID func(token string) (userID string, err error)
}

// eventConn wraps Conn with a specifed event type.
type eventConn struct {
	event string
	conn  *Conn
}
