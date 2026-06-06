package rolls

import (
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/attendance"
)

type Choice string

const (
	ChoiceNeed  Choice = "need"
	ChoiceGreed Choice = "greed"
)

var (
	ErrRollStoreNotFound = errors.New("roll store not found")
	ErrInvalidChoice     = errors.New("invalid roll choice")
)

type RollEvent struct {
	Id           string     `json:"id"`
	Name         string     `json:"name"`
	AttendanceId *string    `json:"attendance_id"`
	EndTime      *time.Time `json:"end_time"`
	Ended        bool       `json:"ended"`

	ChannelId      string `json:"channel_id"`
	EmbedMessageId string `json:"embed_message_id"`
	InputMessageId string `json:"input_message_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RollItem struct {
	Id          string    `json:"id"`
	RollEventId string    `json:"roll_event_id"`
	Name        string    `json:"name"`
	Amount      int       `json:"amount"`
	SortOrder   int       `json:"sort_order"`
	ChannelId   string    `json:"channel_id"`
	MessageId   string    `json:"message_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type RollEntry struct {
	RollEventId string    `json:"roll_event_id"`
	RollItemId  string    `json:"roll_item_id"`
	MemberId    string    `json:"member_id"`
	Choice      Choice    `json:"choice"`
	RollValue   *int      `json:"roll_value"`
	Winner      bool      `json:"winner"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var rollStore rollBackend

func Setup() error {
	return setupRollBackend()
}

func NewEvent(name string, attendanceId *string, endTime *time.Time) *RollEvent {
	now := time.Now().UTC()
	return &RollEvent{
		Id:           xid.New().String(),
		Name:         name,
		AttendanceId: attendanceId,
		EndTime:      endTime,
		Ended:        false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func GetEvent(id string) (*RollEvent, error) {
	if rollStore == nil {
		return nil, ErrRollStoreNotFound
	}
	return rollStore.GetEvent(id)
}

func ListActiveEvents(limit int) ([]*RollEvent, error) {
	if rollStore == nil {
		return nil, ErrRollStoreNotFound
	}
	if limit <= 0 {
		limit = 100
	}
	return rollStore.ListActiveEvents(limit)
}

func (r *RollEvent) Save() error {
	if rollStore == nil {
		return ErrRollStoreNotFound
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	r.UpdatedAt = time.Now().UTC()
	return rollStore.UpsertEvent(r)
}

func (r *RollEvent) End() error {
	if rollStore == nil {
		return ErrRollStoreNotFound
	}
	r.Ended = true
	r.UpdatedAt = time.Now().UTC()
	return rollStore.MarkEventEnded(r.Id, r.UpdatedAt)
}

func (r *RollEvent) Delete() error {
	if rollStore == nil {
		return ErrRollStoreNotFound
	}
	return rollStore.DeleteEvent(r.Id)
}

func (r *RollEvent) CanParticipate(memberId string) (bool, error) {
	if r.AttendanceId == nil || *r.AttendanceId == "" {
		return true, nil
	}

	a, err := attendance.Get(*r.AttendanceId)
	if err != nil {
		return false, err
	}

	return a.HasParticipant(memberId)
}

func (r *RollEvent) Items() ([]*RollItem, error) {
	if rollStore == nil {
		return nil, ErrRollStoreNotFound
	}
	return rollStore.ListItemsByEvent(r.Id)
}

func (r *RollEvent) Entries() ([]*RollEntry, error) {
	if rollStore == nil {
		return nil, ErrRollStoreNotFound
	}
	return rollStore.ListEntriesByEvent(r.Id)
}

func NewItem(rollEventId, name string, amount, sortOrder int) *RollItem {
	if amount <= 0 {
		amount = 1
	}

	return &RollItem{
		Id:          xid.New().String(),
		RollEventId: rollEventId,
		Name:        name,
		Amount:      amount,
		SortOrder:   sortOrder,
		CreatedAt:   time.Now().UTC(),
	}
}

func (i *RollItem) Save() error {
	if rollStore == nil {
		return ErrRollStoreNotFound
	}
	if i.CreatedAt.IsZero() {
		i.CreatedAt = time.Now().UTC()
	}
	return rollStore.UpsertItem(i)
}

func ListItemsByEvent(rollEventId string) ([]*RollItem, error) {
	if rollStore == nil {
		return nil, ErrRollStoreNotFound
	}
	return rollStore.ListItemsByEvent(rollEventId)
}

func NewEntry(rollEventId, rollItemId, memberId string, choice Choice) (*RollEntry, error) {
	if !isValidChoice(choice) {
		return nil, ErrInvalidChoice
	}

	now := time.Now().UTC()
	return &RollEntry{
		RollEventId: rollEventId,
		RollItemId:  rollItemId,
		MemberId:    memberId,
		Choice:      choice,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (e *RollEntry) Save() error {
	if rollStore == nil {
		return ErrRollStoreNotFound
	}
	if !isValidChoice(e.Choice) {
		return ErrInvalidChoice
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	e.UpdatedAt = time.Now().UTC()
	return rollStore.UpsertEntry(e)
}

func ListEntriesByEvent(rollEventId string) ([]*RollEntry, error) {
	if rollStore == nil {
		return nil, ErrRollStoreNotFound
	}
	return rollStore.ListEntriesByEvent(rollEventId)
}

func ListEntriesByItem(rollItemId string) ([]*RollEntry, error) {
	if rollStore == nil {
		return nil, ErrRollStoreNotFound
	}
	return rollStore.ListEntriesByItem(rollItemId)
}

func DeleteEntry(rollItemId, memberId string) error {
	if rollStore == nil {
		return ErrRollStoreNotFound
	}
	return rollStore.DeleteEntry(rollItemId, memberId)
}

func isValidChoice(choice Choice) bool {
	switch choice {
	case ChoiceNeed, ChoiceGreed:
		return true
	default:
		return false
	}
}
