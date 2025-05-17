package service

import "liveclass/internal/global"

func broadcast(message interface{}) error {
	global.Mux.Lock()
	defer global.Mux.Unlock()
	for conn := range global.WsConns {
		err := conn.WriteJSON(message)
		if err != nil {
			return err
		}
	}
	return nil
}
