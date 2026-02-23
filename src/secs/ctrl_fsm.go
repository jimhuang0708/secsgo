// control_fsm.go
package secs

import (
    "errors"
    "fmt"
    "strings"
    "sync"
)

// -------------------- States --------------------

// CtrlMajorState is the top-level control state.
type CtrlMajorState int

const (
    MajorUndefined CtrlMajorState = iota
    MajorOffline
    MajorOnline
)

func (m CtrlMajorState) String() string {
    switch m {
    case MajorUndefined:
        return "UNDEFINED"
    case MajorOffline:
        return "OFFLINE"
    case MajorOnline:
        return "ONLINE"
    default:
        return fmt.Sprintf("CtrlMajorState(%d)", int(m))
    }
}

// CtrlSubState merges OFFLINE CtrlSubStates and ONLINE CtrlSubStates.
type CtrlSubState int

const (
    SubNone CtrlSubState = iota

    // OFFLINE CtrlSubStates
    SubEquipmentOffline
    SubAttemptOnline
    SubHostOffline

    // ONLINE CtrlSubStates
    SubLocal
    SubRemote
)

func (s CtrlSubState) String() string {
    switch s {
    case SubNone:
        return "NONE"
    case SubEquipmentOffline:
        return "EQUIPMENT"
    case SubAttemptOnline:
        return "ATTEMPTONLINE"
    case SubHostOffline:
        return "HOST"
    case SubLocal:
        return "LOCAL"
    case SubRemote:
        return "REMOTE"
    default:
        return fmt.Sprintf("CtrlSubState(%d)", int(s))
    }
}

func (s CtrlSubState) isOfflineSub() bool {
    return s == SubEquipmentOffline || s == SubAttemptOnline || s == SubHostOffline
}

func (s CtrlSubState) isOnlineSub() bool {
    return s == SubLocal || s == SubRemote
}

// ControlState is a complete control state.
type ControlState struct {
    Major CtrlMajorState
    Minor CtrlSubState
}

func (s ControlState) String() string {
    switch s.Major {
    case MajorOffline:
        return fmt.Sprintf("OFFLINE/%s", s.Minor)
    case MajorOnline:
        return fmt.Sprintf("ONLINE/%s", s.Minor)
    case MajorUndefined:
        return "UNDEFINED"
    default:
        return "UNKNOWN"
    }
}

func (s ControlState) Normalize() ControlState {
    switch s.Major {
    case MajorOffline:
        if !s.Minor.isOfflineSub() {
            s.Minor = SubEquipmentOffline
        }
    case MajorOnline:
        if !s.Minor.isOnlineSub() {
            s.Minor = SubLocal
        }
    default:
        s.Minor = SubNone
    }
    return s
}

// -------------------- Events --------------------

// CtrlEventType is used for transition decisions.
type CtrlEventType int

const (
    // Table 3.3
    // #1 Entry into CONTROL state system initialization.
    EvEnterControl CtrlEventType = iota + 1
    // #2 Entry into OFF-LINE state (system initialization).
    //EvEnterOffline
    // #7 Entry into ON-LINE state.
    //EvEnterOnline

    // #3 Operator actuates ON-LINE switch.
    EvOperatorOnlineSwitch
    // #6 Operator actuates OFF-LINE switch (from ONLINE).
    EvOperatorOfflineSwitch

    // #8 Operator sets REMOTE (from LOCAL).
    EvOperatorSetRemote
    // #9 Operator sets LOCAL (from REMOTE).
    EvOperatorSetLocal

    // #10 Host sends "Set OFF-LINE" (S1F15) and equipment accepts.
    EvHostSetOfflineRequest
    // #11 Host requests go ON-LINE (S1F17) and equipment accepts.
    EvHostSetOnlineRequest

    // #4 Attempt ON-LINE failed.
    EvAttemptOnlineFailed
    // #5 Attempt ON-LINE success condition met (e.g. expected S1F2 received).
    EvAttemptOnlineAccepted
)

func (t CtrlEventType) String() string {
    switch t {
    case EvEnterControl:
        return "EnterControl"
    /*case EvEnterOffline:
        return "EnterOffline"
    case EvEnterOnline:
        return "EnterOnline"*/
    case EvOperatorOnlineSwitch:
        return "OperatorOnlineSwitch"
    case EvOperatorOfflineSwitch:
        return "OperatorOfflineSwitch"
    case EvOperatorSetRemote:
        return "OperatorSetRemote"
    case EvOperatorSetLocal:
        return "OperatorSetLocal"
    case EvHostSetOfflineRequest:
        return "HostSetOfflineAccepted"
    case EvHostSetOnlineRequest:
        return "HostGoOnlineAccepted"
    case EvAttemptOnlineFailed:
        return "AttemptOnlineFailed"
    case EvAttemptOnlineAccepted:
        return "AttemptOnlineAccepted"
    default:
        return fmt.Sprintf("CtrlEventType(%d)", int(t))
    }
}

// CtrlEvent carries a type plus reserved payload.
type CtrlEvent struct {
    Type      CtrlEventType
    Parameter interface{} // reserved
}

func NewCtrlEvent(t CtrlEventType, param interface{}) CtrlEvent {
    return CtrlEvent{Type: t, Parameter: param}
}

// -------------------- Hooks (interface) --------------------

type CtrlHooks interface {
    OnTransition(from, to ControlState, ev CtrlEvent, transitionNo int)
    OnInvalidTransition(from ControlState, ev CtrlEvent, reason error)
    OnStateInitialized(state ControlState, ev CtrlEvent, transitionNo int)
}


// -------------------- Config --------------------

type CtrlConfig struct {
    DEFAULT_CTRLMAINSTATE        string
    DEFAULT_CTRLCtrlSubState         string
    DEFAULT_REJECT_CTRLCtrlSubState  string
    DEFAULT_ACCEPT_CTRLCtrlSubState  string
}

// -------------------- Errors --------------------

var (
    ErrInvalidTransition = errors.New("invalid transition for current state")
    ErrBadConfig         = errors.New("bad ctrl config")
)

// -------------------- FSM --------------------

type CtrlFSM struct {
    mu    sync.Mutex
    state ControlState
    cfg   CtrlConfig
    hooks CtrlHooks
}

func CreateCtrlFSM(cfg CtrlConfig, hooks CtrlHooks) (*CtrlFSM, error) {

    // Validate config strings early (so runtime transitions don't blow up unexpectedly).
    if _, err := parseCtrlMajorState(cfg.DEFAULT_CTRLMAINSTATE); err != nil {
        return nil, fmt.Errorf("%w: DEFAULT_CTRLMAINSTATE: %v", ErrBadConfig, err)
    }
    if _, err := parseCtrlSubState(cfg.DEFAULT_CTRLCtrlSubState); err != nil {
        return nil, fmt.Errorf("%w: DEFAULT_CTRLCtrlSubState: %v", ErrBadConfig, err)
    }
    if _, err := parseCtrlSubState(cfg.DEFAULT_REJECT_CTRLCtrlSubState); err != nil {
        return nil, fmt.Errorf("%w: DEFAULT_REJECT_CTRLCtrlSubState: %v", ErrBadConfig, err)
    }
    if _, err := parseCtrlSubState(cfg.DEFAULT_ACCEPT_CTRLCtrlSubState); err != nil {
        return nil, fmt.Errorf("%w: DEFAULT_ACCEPT_CTRLCtrlSubState: %v", ErrBadConfig, err)
    }

    init := ControlState{Major: MajorUndefined, Minor: SubNone}.Normalize()

    return &CtrlFSM{
        state: init,
        cfg:   cfg,
        hooks: hooks,
    }, nil
}

func (f *CtrlFSM) State() ControlState {
    f.mu.Lock()
    defer f.mu.Unlock()
    return f.state
}

func (f *CtrlFSM) Emit(ev CtrlEvent) error {
    f.mu.Lock()
    defer f.mu.Unlock()

    from := f.state
    to := from
    transitionNo := 0

    apply := func(newState ControlState, tNo int) {
        to = newState.Normalize()
        transitionNo = tNo
    }

    defMajor, _ := parseCtrlMajorState(f.cfg.DEFAULT_CTRLMAINSTATE)
    defSub, _ := parseCtrlSubState(f.cfg.DEFAULT_CTRLCtrlSubState)
    rejectSub, _ := parseCtrlSubState(f.cfg.DEFAULT_REJECT_CTRLCtrlSubState)
    acceptSub, _ := parseCtrlSubState(f.cfg.DEFAULT_ACCEPT_CTRLCtrlSubState)

    // -------------------- Transition Table (Table 3.3) --------------------
    switch ev.Type {

    // #1 (Undefined) Entry into CONTROL state system initialization.
    case EvEnterControl:
        if from.Major != MajorUndefined {
            return f.invalid(from, ev, fmt.Errorf("%w: EnterControl only allowed from UNDEFINED", ErrInvalidTransition))
        }
        apply(ControlState{Major: defMajor, Minor: defSub}, 1)
        f.state = to
        f.hooks.OnStateInitialized(f.state, ev, transitionNo)
        f.hooks.OnTransition(from, to, ev, transitionNo)
        return nil

    // #2 (Undefined) Entry into OFF-LINE state.
    /*case EvEnterOffline:
        if from.Major != MajorUndefined {
            return f.invalid(from, ev, fmt.Errorf("%w: EnterOffline only allowed from UNDEFINED", ErrInvalidTransition))
        }
        apply(ControlState{Major: MajorOffline, Minor: defSub}, 2)
        f.state = to
        f.hooks.OnStateInitialized(f.state, ev, transitionNo)
        f.hooks.OnTransition(from, to, ev, transitionNo)
        return nil
     */
    // #7 (Undefined) Entry into ON-LINE state.
    /*case EvEnterOnline:
        if from.Major != MajorUndefined {
            return f.invalid(from, ev, fmt.Errorf("%w: EnterOnline only allowed from UNDEFINED", ErrInvalidTransition))
        }
        // If DEFAULT_CTRLCtrlSubState is offline-like, Normalize() will force LOCAL.
        apply(ControlState{Major: MajorOnline, Minor: defSub}, 7)
        f.state = to
        f.hooks.OnStateInitialized(f.state, ev, transitionNo)
        f.hooks.OnTransition(from, to, ev, transitionNo)
        return nil
     */
    // #3 EQUIPMENT_OFF-LINE + Operator actuates ON-LINE switch => ATTEMPT_ON-LINE
    case EvOperatorOnlineSwitch:
        if from.Major == MajorOffline && from.Minor == SubEquipmentOffline {
            apply(ControlState{Major: MajorOffline, Minor: SubAttemptOnline}, 3)
        } else if from.Major == MajorOffline && from.Minor == SubHostOffline {
            return f.invalid(from, ev, fmt.Errorf("%w: operator ON-LINE denied in HOST_OFFLINE", ErrInvalidTransition))
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: operator ON-LINE only valid in OFFLINE/EQUIPMENT_OFFLINE", ErrInvalidTransition))
        }

    // #4 ATTEMPT_ON-LINE + failure => DEFAULT_REJECT_CTRLCtrlSubState (offline-like)
    case EvAttemptOnlineFailed:
        if from.Major == MajorOffline && from.Minor == SubAttemptOnline {
            apply(ControlState{Major: MajorOffline, Minor: rejectSub}, 4)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: AttemptOnlineFailed only valid in OFFLINE/ATTEMPT_ONLINE", ErrInvalidTransition))
        }

    // #5 ATTEMPT_ON-LINE + accepted condition => DEFAULT_ACCEPT_CTRLCtrlSubState (online-like)
    case EvAttemptOnlineAccepted:
        if from.Major == MajorOffline && from.Minor == SubAttemptOnline {
            // IMPORTANT: acceptSub is defined as "Attemponline 成功後的 CtrlSubState"
            // It should normally be LOCAL or REMOTE. Normalize() will enforce online CtrlSubState if MajorOnline.
            apply(ControlState{Major: MajorOnline, Minor: acceptSub}, 5)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: AttemptOnlineAccepted only valid in OFFLINE/ATTEMPT_ONLINE", ErrInvalidTransition))
        }

    // #6 ON-LINE + Operator actuates OFF-LINE switch => EQUIPMENT_OFF-LINE
    case EvOperatorOfflineSwitch:
        if from.Major == MajorOnline {
            apply(ControlState{Major: MajorOffline, Minor: SubEquipmentOffline}, 6)
        } else if from.Major == MajorOffline &&  from.Minor == SubHostOffline {
            apply(ControlState{Major: MajorOffline, Minor: SubEquipmentOffline}, 12)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: operator OFF-LINE invalid ", ErrInvalidTransition))
        }

    // #8 LOCAL + operator sets REMOTE => REMOTE
    case EvOperatorSetRemote:
        if from.Major == MajorOnline && from.Minor == SubLocal {
            apply(ControlState{Major: MajorOnline, Minor: SubRemote}, 8)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: SetRemote only valid in ONLINE/LOCAL", ErrInvalidTransition))
        }

    // #9 REMOTE + operator sets LOCAL => LOCAL
    case EvOperatorSetLocal:
        if from.Major == MajorOnline && from.Minor == SubRemote {
            apply(ControlState{Major: MajorOnline, Minor: SubLocal}, 9)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: SetLocal only valid in ONLINE/REMOTE", ErrInvalidTransition))
        }

    // #10 ON-LINE + host accepts Set OFF-LINE => HOST_OFF-LINE
    case EvHostSetOfflineRequest:
        if from.Major == MajorOnline {
            apply(ControlState{Major: MajorOffline, Minor: SubHostOffline}, 10)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: HostSetOfflineAccepted only valid from ONLINE", ErrInvalidTransition))
        }

    // #11 HOST_OFF-LINE + host accepts go ON-LINE => ON-LINE (DEFAULT_ACCEPT_CTRLCtrlSubState is used)
    case EvHostSetOnlineRequest:
        if from.Major == MajorOffline && from.Minor == SubHostOffline {
            apply(ControlState{Major: MajorOnline, Minor: acceptSub}, 11)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: HostGoOnlineAccepted only valid in OFFLINE/HOST_OFFLINE", ErrInvalidTransition))
        }

    default:
        return f.invalid(from, ev, fmt.Errorf("%w: unknown event type %v", ErrInvalidTransition, ev.Type))
    }

    if transitionNo == 0 {
        return f.invalid(from, ev, fmt.Errorf("%w: no transition fired", ErrInvalidTransition))
    }

    f.state = to
    f.hooks.OnTransition(from, to, ev, transitionNo)
    return nil
}

func (f *CtrlFSM) invalid(from ControlState, ev CtrlEvent, reason error) error {
    f.hooks.OnInvalidTransition(from, ev, reason)
    return reason
}

// -------------------- Parsing helpers (string -> enum) --------------------

func parseCtrlMajorState(s string) (CtrlMajorState, error) {
    v := strings.ToUpper(strings.TrimSpace(s))
    switch v {
    case "OFFLINE":
        return MajorOffline, nil
    case "ONLINE":
        return MajorOnline, nil
    case "UNDEFINED", "":
        return MajorUndefined, nil
    default:
        return MajorUndefined, fmt.Errorf("unsupported main state %q (use OFFLINE/ONLINE)", s)
    }
}

func parseCtrlSubState(s string) (CtrlSubState, error) {
    v := strings.ToUpper(strings.TrimSpace(s))
    switch v {
    case "NONE", "":
        return SubNone, nil
    case "EQUIPMENT":
        return SubEquipmentOffline, nil
    case "ATTEMPTONLINE":
        return SubAttemptOnline, nil
    case "HOST":
        return SubHostOffline, nil
    case "LOCAL":
        return SubLocal, nil
    case "REMOTE":
        return SubRemote, nil
    default:
        return SubNone, fmt.Errorf("unsupported sub state %q (use EQUIPMENT_OFFLINE/ATTEMPT_ONLINE/HOST_OFFLINE/LOCAL/REMOTE)", s)
    }
}
