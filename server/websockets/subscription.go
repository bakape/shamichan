package websockets

// Response to a clients synchronisation, resynchronisation or thread swicth
// request
type syncResponse struct {
	ID      string   `json:"id"`
	Sync    int64    `json:"sync"`
	Backlog []string `json:"backlog"`
}

// // Sends a response to the client's synchronisation request
// func (s *subscription) sendSyncResponse(req subRequest) {
// 	msg := syncResponse{
// 		ID:   req.client.ID,
// 		Sync: s.counter,
// 	}
//
// 	// Send to the client, mesasges it is behind on
// 	diff := int(s.counter - req.counter)
// 	if diff > 0 {
// 		msg.Backlog = make([]string, diff)
// 		toSend := s.log[len(s.log)-diff:]
// 		for i := 0; i < diff; i++ {
// 			msg.Backlog[i] = string(toSend[i])
// 		}
//
// 		data, err := json.Marshal(&msg)
// 		if err != nil {
// 			req.client.logError(errors.New("Error encoding backlog"))
// 			return
// 		}
// 		req.client.Send(data)
// 	}
// }
