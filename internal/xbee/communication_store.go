package xbee

import (
	"fmt"
	"sync"
	"time"
)

// CommandLog captures a single command transmitted by the ground station.
type CommandLog struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Command      string    `json:"command"`
	RawData      []byte    `json:"rawData"`
	RemoteAddr64 string    `json:"remoteAddr64"`
	RemoteAddr16 string    `json:"remoteAddr16"`
	MissionID    string    `json:"missionId,omitempty"`
}

// ResponseLog captures any frame received from the remote XBee except telemetry.
type ResponseLog struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	ResponseType string    `json:"responseType"`
	RawData      []byte    `json:"rawData"`
	ParsedData   string    `json:"parsedData"`
	RemoteAddr64 string    `json:"remoteAddr64"`
	RemoteAddr16 string    `json:"remoteAddr16"`
	ClusterID    uint16    `json:"clusterId,omitempty"`
	ProfileID    uint16    `json:"profileId,omitempty"`
	MissionID    string    `json:"missionId,omitempty"`
}

// CommunicationSnapshot returns a read-only copy of the command/response history.
type CommunicationSnapshot struct {
	Commands       []CommandLog  `json:"commands"`
	Responses      []ResponseLog `json:"responses"`
	TotalCommands  int           `json:"totalCommands"`
	TotalResponses int           `json:"totalResponses"`
}

type communicationStore struct {
	mu        sync.RWMutex
	commands  []CommandLog
	responses []ResponseLog
	maxItems  int
}

func newCommunicationStore(maxItems int) *communicationStore {
	if maxItems <= 0 {
		maxItems = 10000
	}

	return &communicationStore{
		commands:  make([]CommandLog, 0, maxItems/2),
		responses: make([]ResponseLog, 0, maxItems/2),
		maxItems:  maxItems,
	}
}

func (s *communicationStore) recordCommand(command, missionID string, rawData []byte, remoteAddr64, remoteAddr16 string) CommandLog {
	entry := CommandLog{
		ID:           fmt.Sprintf("CMD_%d", time.Now().UnixNano()),
		Timestamp:    time.Now(),
		Command:      command,
		RawData:      cloneBytes(rawData),
		RemoteAddr64: remoteAddr64,
		RemoteAddr16: remoteAddr16,
		MissionID:    missionID,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.commands = append(s.commands, entry)
	if len(s.commands) > s.maxItems {
		s.commands = s.commands[len(s.commands)-s.maxItems:]
	}

	return entry
}

func (s *communicationStore) recordResponse(responseType string, missionID string, rawData []byte, parsedData, remoteAddr64, remoteAddr16 string, clusterID, profileID uint16) ResponseLog {
	entry := ResponseLog{
		ID:           fmt.Sprintf("RSP_%d", time.Now().UnixNano()),
		Timestamp:    time.Now(),
		ResponseType: responseType,
		RawData:      cloneBytes(rawData),
		ParsedData:   parsedData,
		RemoteAddr64: remoteAddr64,
		RemoteAddr16: remoteAddr16,
		ClusterID:    clusterID,
		ProfileID:    profileID,
		MissionID:    missionID,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.responses = append(s.responses, entry)
	if len(s.responses) > s.maxItems {
		s.responses = s.responses[len(s.responses)-s.maxItems:]
	}

	return entry
}

func (s *communicationStore) snapshot() CommunicationSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	commands := make([]CommandLog, len(s.commands))
	copy(commands, s.commands)

	responses := make([]ResponseLog, len(s.responses))
	copy(responses, s.responses)

	return CommunicationSnapshot{
		Commands:       commands,
		Responses:      responses,
		TotalCommands:  len(commands),
		TotalResponses: len(responses),
	}
}

func (s *communicationStore) latestCommands() []CommandLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]CommandLog, len(s.commands))
	copy(out, s.commands)
	return out
}

func (s *communicationStore) latestResponses() []ResponseLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ResponseLog, len(s.responses))
	copy(out, s.responses)
	return out
}

func cloneBytes(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}
	out := make([]byte, len(in))
	copy(out, in)
	return out
}
