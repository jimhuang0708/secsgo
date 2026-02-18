// control_fsm.go
package secs

import (
    "errors"
    "fmt"
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

// OfflineSubstate applies only when MajorOffline.
type OfflineSubstate int


const (
    OffSubNone OfflineSubstate = iota
    OffEquipmentOffline
    OffAttemptOnline
    OffHostOffline
)

func (s OfflineSubstate) String() string {
    switch s {
    case OffSubNone:
        return "NONE"
    case OffEquipmentOffline:
        return "EQUIPMENT_OFFLINE"
    case OffAttemptOnline:
        return "ATTEMPT_ONLINE"
    case OffHostOffline:
        return "HOST_OFFLINE"
    default:
        return fmt.Sprintf("OfflineSubstate(%d)", int(s))
    }
}

// OnlineSubstate applies only when MajorOnline.
type OnlineSubstate int

const (
    OnSubNone OnlineSubstate = iota
    OnLocal
    OnRemote
)

func (s OnlineSubstate) String() string {
    switch s {
    case OnSubNone:
        return "NONE"
    case OnLocal:
        return "LOCAL"
    case OnRemote:
        return "REMOTE"
    default:
        return fmt.Sprintf("OnlineSubstate(%d)", int(s))
    }
}

// ControlState is a complete control state.
type ControlState struct {
    Major   CtrlMajorState
    Offline OfflineSubstate
    Online  OnlineSubstate
}

func (s ControlState) String() string {
    switch s.Major {
    case MajorOffline:
        return fmt.Sprintf("OFFLINE/%s", s.Offline)
    case MajorOnline:
        return fmt.Sprintf("ONLINE/%s", s.Online)
    case MajorUndefined:
        return "UNDEFINED"
    default:
        return "UNKNOWN"
    }
}

func (s ControlState) Normalize() ControlState {
    switch s.Major {
    case MajorOffline:
        s.Online = OnSubNone
        if s.Offline == OffSubNone {
            s.Offline = OffEquipmentOffline
        }
    case MajorOnline:
        s.Offline = OffSubNone
        if s.Online == OnSubNone {
            s.Online = OnLocal
        }
    default:
        s.Offline = OffSubNone
        s.Online = OnSubNone
    }
    return s
}

// -------------------- Events --------------------

// EventType is used for transition decision.
type EventType int

const (
    // Table 3.3
    // #1 Entry into CONTROL state system initialization.
    EvEnterControl EventType = iota + 1
    // #2 Entry into OFF-LINE state (system initialization).
    EvEnterOffline
    // #7 Entry into ON-LINE state.
    EvEnterOnline

    // #3 Operator actuates ON-LINE switch.
    EvOperatorOnlineSwitch
    // #6 Operator actuates OFF-LINE switch (from ONLINE).
    EvOperatorOfflineSwitch

    // #8 Operator sets REMOTE (from LOCAL).
    EvOperatorSetRemote
    // #9 Operator sets LOCAL (from REMOTE).
    EvOperatorSetLocal

    // #10 Host sends "Set OFF-LINE" (S1F15) and equipment accepts.
    EvHostSetOfflineAccepted
    // #11 Host requests go ON-LINE (S1F17) and equipment accepts.
    EvHostGoOnlineAccepted

    // #4 Attempt ON-LINE times out / comm failure / bad S1F2 / etc.
    EvAttemptOnlineFailed
    // #5 Equipment receives expected S1F2 while attempting on-line.
    EvAttemptOnlineS1F2Received
)

func (t EventType) String() string {
    switch t {
    case EvEnterControl:
        return "EnterControl"
    case EvEnterOffline:
        return "EnterOffline"
    case EvEnterOnline:
        return "EnterOnline"
    case EvOperatorOnlineSwitch:
        return "OperatorOnlineSwitch"
    case EvOperatorOfflineSwitch:
        return "OperatorOfflineSwitch"
    case EvOperatorSetRemote:
        return "OperatorSetRemote"
    case EvOperatorSetLocal:
        return "OperatorSetLocal"
    case EvHostSetOfflineAccepted:
        return "HostSetOfflineAccepted(S1F15)"
    case EvHostGoOnlineAccepted:
        return "HostGoOnlineAccepted(S1F17)"
    case EvAttemptOnlineFailed:
        return "AttemptOnlineFailed"
    case EvAttemptOnlineS1F2Received:
        return "AttemptOnlineS1F2Received(S1F2)"
    default:
        return fmt.Sprintf("EventType(%d)", int(t))
    }
}

// Event carries a type plus optional payload.
type CtrlFSMEvent struct {
    Type      EventType
    Parameter interface{}
}

// -------------------- Hooks --------------------

// Hooks lets you attach side effects (send messages, publish events, logs, etc.)
type Hooks struct {
    // OnTransition is called after a transition is applied.
    OnTransition func(from, to ControlState, ev CtrlFSMEvent, transitionNo int)

    // GEM-ish helper callbacks (optional)
    OnEnterAttemptOnline     func(ev CtrlFSMEvent)
    OnEnterOffline           func(sub OfflineSubstate, ev CtrlFSMEvent)
    OnOnlineSubstateChanged  func(sub OnlineSubstate, ev CtrlFSMEvent)
    OnInvalidTransition      func(from ControlState, ev CtrlFSMEvent, reason error)
    OnStateInitialized       func(state ControlState, ev CtrlFSMEvent, transitionNo int)
}

// -------------------- Errors --------------------

var (
    ErrInvalidTransition = errors.New("invalid transition for current state")
)

// -------------------- FSM --------------------

type FSM struct {
    mu    sync.Mutex
    state ControlState
    hooks Hooks
}

type CtrlFSMCfg struct {
    // For #1 (EnterControl) "CONTROL (Substate conditional on configuration)"
    // If true -> initial enters ONLINE; otherwise OFFLINE.
    DefaultOnline bool

    // For #2 (EnterOffline) default offline substate.
    DefaultOfflineSub OfflineSubstate

    // For #7 (EnterOnline) default online substate.
    DefaultOnlineSub OnlineSubstate
}

func NewFSM(cfg CtrlFSMCfg, hooks Hooks) *FSM {
    off := cfg.DefaultOfflineSub
    if off == OffSubNone {
        off = OffEquipmentOffline
    }
    on := cfg.DefaultOnlineSub
    if on == OnSubNone {
        on = OnLocal
    }

    init := ControlState{Major: MajorUndefined, Offline: OffSubNone, Online: OnSubNone}.Normalize()

    // store defaults in FSM via hooks closure? We'll keep cfg in struct for clarity.
    return &FSM{
        state: init,
        hooks: hooks,
    }
}

// State returns current state snapshot.
func (f *FSM) State() ControlState {
    f.mu.Lock()
    defer f.mu.Unlock()
    return f.state
}

// Emit applies an event to the FSM.
// Event.Parameter is carried to hooks for side effects.
func (f *FSM) Emit(cfg CtrlFSMCfg, ev CtrlFSMEvent) error {
    f.mu.Lock()
    defer f.mu.Unlock()

    from := f.state
    to := from
    var transitionNo int

    // Normalize cfg defaults
    defOff := cfg.DefaultOfflineSub
    if defOff == OffSubNone {
        defOff = OffEquipmentOffline
    }
    defOn := cfg.DefaultOnlineSub
    if defOn == OnSubNone {
        defOn = OnLocal
    }

    apply := func(newState ControlState, tNo int) {
        to = newState.Normalize()
        transitionNo = tNo
    }

    // -------------------- Transition Table (Table 3.3) --------------------
    switch ev.Type {

    // #1 (Undefined) Entry into CONTROL state system initialization.
    case EvEnterControl:
        if from.Major != MajorUndefined {
            return f.invalid(from, ev, fmt.Errorf("%w: EnterControl only allowed from UNDEFINED", ErrInvalidTransition))
        }
        if cfg.DefaultOnline {
            apply(ControlState{Major: MajorOnline, Online: defOn}, 1)
        } else {
            apply(ControlState{Major: MajorOffline, Offline: defOff}, 1)
        }

    // #2 (Undefined) Entry into OFF-LINE state.
    case EvEnterOffline:
        if from.Major != MajorUndefined {
            return f.invalid(from, ev, fmt.Errorf("%w: EnterOffline only allowed from UNDEFINED", ErrInvalidTransition))
        }
        apply(ControlState{Major: MajorOffline, Offline: defOff}, 2)

    // #7 (Undefined) Entry into ON-LINE state.
    case EvEnterOnline:
        if from.Major != MajorUndefined {
            return f.invalid(from, ev, fmt.Errorf("%w: EnterOnline only allowed from UNDEFINED", ErrInvalidTransition))
        }
        apply(ControlState{Major: MajorOnline, Online: defOn}, 7)

    // #3 EQUIPMENT_OFF-LINE + Operator actuates ON-LINE switch => ATTEMPT_ON-LINE
    case EvOperatorOnlineSwitch:
        if from.Major == MajorOffline && from.Offline == OffEquipmentOffline {
            apply(ControlState{Major: MajorOffline, Offline: OffAttemptOnline}, 3)
        } else if from.Major == MajorOffline && from.Offline == OffHostOffline {
            // spec text: in HOST_OFFLINE operator may request ON-LINE but must be denied.
            return f.invalid(from, ev, fmt.Errorf("%w: operator ON-LINE denied in HOST_OFFLINE", ErrInvalidTransition))
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: operator ON-LINE only valid in OFFLINE/EQUIPMENT_OFFLINE", ErrInvalidTransition))
        }

    // #4 ATTEMPT_ON-LINE + failure => new state conditional on configuration (comm fail -> equipment/host offline)
    case EvAttemptOnlineFailed:
        if from.Major == MajorOffline && from.Offline == OffAttemptOnline {
            // Common practice: if comm not established -> EQUIPMENT_OFFLINE; if host forced -> HOST_OFFLINE.
            // Use cfg.DefaultOfflineSub to decide, or allow caller to put desired target in ev.Parameter.
            if target, ok := ev.Parameter.(OfflineSubstate); ok && (target == OffEquipmentOffline || target == OffHostOffline) {
                apply(ControlState{Major: MajorOffline, Offline: target}, 4)
            } else {
                apply(ControlState{Major: MajorOffline, Offline: defOff}, 4)
            }
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: AttemptOnlineFailed only valid in OFFLINE/ATTEMPT_ONLINE", ErrInvalidTransition))
        }

    // #5 ATTEMPT_ON-LINE + receives expected S1F2 => ON-LINE
    case EvAttemptOnlineS1F2Received:
        if from.Major == MajorOffline && from.Offline == OffAttemptOnline {
            apply(ControlState{Major: MajorOnline, Online: defOn}, 5)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: AttemptOnlineS1F2Received only valid in OFFLINE/ATTEMPT_ONLINE", ErrInvalidTransition))
        }

    // #6 ON-LINE + Operator actuates OFF-LINE switch => EQUIPMENT_OFF-LINE
    case EvOperatorOfflineSwitch:
        if from.Major == MajorOnline {
            apply(ControlState{Major: MajorOffline, Offline: OffEquipmentOffline}, 6)
        } else if from.Major == MajorOffline {
            // already offline: no-op or invalid. Choose invalid for clarity.
            return f.invalid(from, ev, fmt.Errorf("%w: operator OFF-LINE only meaningful from ONLINE", ErrInvalidTransition))
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: operator OFF-LINE invalid from UNDEFINED", ErrInvalidTransition))
        }

    // #8 LOCAL + operator sets REMOTE => REMOTE
    case EvOperatorSetRemote:
        if from.Major == MajorOnline && from.Online == OnLocal {
            apply(ControlState{Major: MajorOnline, Online: OnRemote}, 8)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: SetRemote only valid in ONLINE/LOCAL", ErrInvalidTransition))
        }

    // #9 REMOTE + operator sets LOCAL => LOCAL
    case EvOperatorSetLocal:
        if from.Major == MajorOnline && from.Online == OnRemote {
            apply(ControlState{Major: MajorOnline, Online: OnLocal}, 9)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: SetLocal only valid in ONLINE/REMOTE", ErrInvalidTransition))
        }

    // #10 ON-LINE + host accepts Set OFF-LINE (S1F15) => HOST_OFF-LINE
    case EvHostSetOfflineAccepted:
        if from.Major == MajorOnline {
            apply(ControlState{Major: MajorOffline, Offline: OffHostOffline}, 10)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: HostSetOfflineAccepted only valid from ONLINE", ErrInvalidTransition))
        }

    // #11 HOST_OFF-LINE + host accepts go ON-LINE (S1F17) => ON-LINE (transition 7 entry to online)
    case EvHostGoOnlineAccepted:
        if from.Major == MajorOffline && from.Offline == OffHostOffline {
            apply(ControlState{Major: MajorOnline, Online: defOn}, 11)
        } else {
            return f.invalid(from, ev, fmt.Errorf("%w: HostGoOnlineAccepted only valid in OFFLINE/HOST_OFFLINE", ErrInvalidTransition))
        }

    default:
        return f.invalid(from, ev, fmt.Errorf("%w: unknown event type %v", ErrInvalidTransition, ev.Type))
    }

    // If no state changed (shouldn't happen here), treat as invalid.
    if transitionNo == 0 {
        return f.invalid(from, ev, fmt.Errorf("%w: no transition fired", ErrInvalidTransition))
    }

    // Apply state
    f.state = to

    // -------------------- Hooks / Side effects --------------------
    // Initialize hook (optional - same as transition hook but clearer)
    if f.hooks.OnStateInitialized != nil && (ev.Type == EvEnterControl || ev.Type == EvEnterOffline || ev.Type == EvEnterOnline) {
        f.hooks.OnStateInitialized(f.state, ev, transitionNo)
    }

    // Enter offline hook
    if from.Major != MajorOffline && to.Major == MajorOffline {
        if f.hooks.OnEnterOffline != nil {
            f.hooks.OnEnterOffline(to.Offline, ev)
        }
    }

    // Enter attempt online hook
    if to.Major == MajorOffline && to.Offline == OffAttemptOnline && !(from.Major == MajorOffline && from.Offline == OffAttemptOnline) {
        if f.hooks.OnEnterAttemptOnline != nil {
            f.hooks.OnEnterAttemptOnline(ev)
        }
    }

    // Online substate change hook
    if to.Major == MajorOnline {
        if from.Major != MajorOnline || from.Online != to.Online {
            if f.hooks.OnOnlineSubstateChanged != nil {
                f.hooks.OnOnlineSubstateChanged(to.Online, ev)
            }
        }
    }

    // Transition hook
    if f.hooks.OnTransition != nil {
        f.hooks.OnTransition(from, to, ev, transitionNo)
    }

    return nil
}

func (f *FSM) invalid(from ControlState, ev CtrlFSMEvent , reason error) error {
    if f.hooks.OnInvalidTransition != nil {
        f.hooks.OnInvalidTransition(from, ev, reason)
    }
    return reason
}

// -------------------- Helpers --------------------

// Convenience constructors
func NewEvent(t EventType, param interface{}) CtrlFSMEvent {
    return CtrlFSMEvent{Type: t, Parameter: param}
}

// DefaultCtrlFSMConfig returns a sensible default mapping:
// - default enter offline => EQUIPMENT_OFFLINE
// - default enter online => LOCAL
func DefaultCtrlFSMConfig() CtrlFSMCfg {
    return CtrlFSMCfg{
        DefaultOnline:     false,
        DefaultOfflineSub: OffEquipmentOffline,
        DefaultOnlineSub:  OnLocal,
    }
}
