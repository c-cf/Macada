package domain

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

const (
	PrefixAgent       = "agent_"
	PrefixSession     = "sesn_"
	PrefixEnvironment = "env_"
	PrefixEvent       = "sevt_"
	PrefixLLMLog      = "llml_"
	PrefixSkill       = "skil_"
	PrefixWorkspace   = "ws_"
	PrefixAPIKey      = "key_"
	PrefixUser        = "usr_"
	PrefixFile        = "file_"
	PrefixResource    = "res_"
)

func NewID(prefix string) string {
	return prefix + ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

func NewAgentID() string       { return NewID(PrefixAgent) }
func NewSessionID() string     { return NewID(PrefixSession) }
func NewEnvironmentID() string { return NewID(PrefixEnvironment) }
func NewEventID() string       { return NewID(PrefixEvent) }
func NewLLMLogID() string      { return NewID(PrefixLLMLog) }
func NewSkillID() string       { return NewID(PrefixSkill) }
func NewWorkspaceID() string   { return NewID(PrefixWorkspace) }
func NewAPIKeyID() string      { return NewID(PrefixAPIKey) }
func NewUserID() string        { return NewID(PrefixUser) }
func NewFileID() string        { return NewID(PrefixFile) }
func NewResourceID() string    { return NewID(PrefixResource) }
