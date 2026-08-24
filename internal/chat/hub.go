package chat

import "sync"

// EventType — тип события, которое пушится клиенту через SSE.
type EventType string

const (
	EventMessageCreated EventType = "message_created"
	EventMessageDeleted EventType = "message_deleted"
	EventTyping         EventType = "typing"
	EventReadReceipt    EventType = "read_receipt"
	EventChatDeleted    EventType = "chat_deleted"
	// EventChatCreated — новый чат создан или в существующий добавили
	// участника. Данные — только chatId: у каждого участника свой Role/
	// UnreadCount/LastReadMessageID (см. ChatWithMeta), поэтому проще
	// попросить фронт перезапросить список, чем гонять персонализированный
	// снапшот через один и тот же ивент на всех.
	EventChatCreated EventType = "chat_created"
	// EventParticipantRemoved — из группового чата убрали участника (или он
	// вышел сам). Шлётся ОСТАВШИМСЯ участникам, чтобы обновить состав в уже
	// открытом треде; самому удалённому вместо этого уходит EventChatDeleted
	// (чат у него просто пропадает — RemoveParticipant в service.go).
	EventParticipantRemoved EventType = "participant_removed"
)

// Event — конверт события для одного SSE-сообщения.
type Event struct {
	Type EventType `json:"type"`
	Data any       `json:"data"`
}

// Hub — in-memory реестр активных SSE-подключений: userID -> набор каналов
// (у одного пользователя может быть открыто несколько вкладок/устройств).
//
// Работает только в рамках ОДНОГО процесса — поэтому prefork в cmd/api.go
// отключён (см. комментарий там): при нескольких ОС-процессах хаб одного
// из них не видит соединения, принятые другим, и часть участников чата
// не получала бы события.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[chan Event]struct{}
	// viewing — userID -> id чата, который сейчас открыт на фронте (см.
	// SetViewing/ClearViewing/IsViewing). Нужен, чтобы не слать VK-дубликат
	// уведомления, пока человек и так сидит в этом чате в приложении.
	viewing map[string]string
}

func NewHub() *Hub {
	return &Hub{
		conns:   make(map[string]map[chan Event]struct{}),
		viewing: make(map[string]string),
	}
}

// Subscribe регистрирует новое SSE-соединение пользователя и возвращает
// канал, в который будут падать события для него. Буфер небольшой —
// если читатель не успевает (клиент завис), новые события для него просто
// теряются (см. SendToUser), а не блокируют отправителя/остальных подписчиков.
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

// Unsubscribe снимает соединение с регистрации. Обязательно вызывать через
// defer сразу после Subscribe, иначе реестр будет расти бесконечно.
func (h *Hub) Unsubscribe(userID string, ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if set, ok := h.conns[userID]; ok {
		delete(set, ch)
		if len(set) == 0 {
			delete(h.conns, userID)
			// Соединений (вкладок) не осталось — значит, ничего физически
			// не открыто, viewing протух сам по себе (закрытие вкладки/сеть
			// упала — явный ClearViewing от фронта в этом случае не придёт).
			delete(h.viewing, userID)
		}
	}
	close(ch)
}

// SetViewing/ClearViewing/IsViewing — см. комментарий у поля viewing.
func (h *Hub) SetViewing(userID, chatID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.viewing[userID] = chatID
}

func (h *Hub) ClearViewing(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.viewing, userID)
}

func (h *Hub) IsViewing(userID, chatID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.viewing[userID] == chatID
}

// SendToUser рассылает событие во все открытые соединения пользователя.
// Не блокирует: если у какого-то подписчика буфер полон, событие для него
// пропускается (клиент узнает актуальное состояние при следующем запросе
// истории/списка — SSE тут дополнение к REST, а не единственный источник
// правды).
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

// Broadcast рассылает событие сразу нескольким пользователям (участникам чата).
func (h *Hub) Broadcast(userIDs []string, event Event) {
	for _, id := range userIDs {
		h.SendToUser(id, event)
	}
}
