// Package events diffuse des événements temps réel (progression d'analyse…) aux navigateurs via SSE.
package events

import (
	"encoding/json"
	"sync"
)

// Event : un événement nommé avec une charge JSON.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Hub : diffusion vers des abonnés. Un abonné lent perd des événements plutôt que de bloquer l'émetteur.
type Hub struct {
	mu   sync.Mutex
	subs map[chan []byte]bool
}

// NewHub crée un hub.
func NewHub() *Hub { return &Hub{subs: map[chan []byte]bool{}} }

// Subscribe enregistre un abonné ; appeler la fonction renvoyée pour se désabonner.
func (h *Hub) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 32)
	h.mu.Lock()
	h.subs[ch] = true
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if h.subs[ch] {
			delete(h.subs, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
}

// Publish envoie un événement à tous les abonnés.
func (h *Hub) Publish(typ string, data any) {
	b, err := json.Marshal(Event{typ, data})
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- b:
		default:
		}
	}
}
