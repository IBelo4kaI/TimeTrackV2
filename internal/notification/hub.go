package notification

import "sync"

// Event — конверт события для одного SSE-сообщения. Тот же паттерн, что
// chat.Hub, но отдельный: notification не должен зависеть от chat, а chat
// (наоборот) теперь зависит от notification (см. service.go).
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Hub — in-memory реестр активных SSE-подключений: userID -> набор каналов
// (несколько вкладок/устройств у одного пользователя). Как и chat.Hub,
// работает только в рамках одного процесса — prefork в cmd/api.go по этой
// же причине отключён.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string]map[chan Event]struct{})}
}

func (h *Hub) Subscribe(userID string) chan Event {
	ch := make(chan Event, 16)

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conns[userID] == nil {
		h.conns[userID] = make(map[chan Event]struct{})
	}
	h.conns[userID][ch] = struct{}{}

	return ch
}

func (h *Hub) Unsubscribe(userID string, ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if set, ok := h.conns[userID]; ok {
		delete(set, ch)
		if len(set) == 0 {
			delete(h.conns, userID)
		}
	}
	close(ch)
}

// SendToUser не блокирует: если буфер подписчика полон, событие для него
// пропускается (при следующем открытии бела/перезагрузке страницы список
// подтянется через REST — SSE тут дополнение, а не единственный источник).
func (h *Hub) SendToUser(userID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.conns[userID] {
		select {
		case ch <- event:
		default:
		}
	}
}
