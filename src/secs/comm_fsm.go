// comm_fsm_and.go
//
// Spec-driven Communications State ComFSM (SEMI E5 / SECS-I style) with
// correct Harel AND-state semantics for NOT COMMUNICATING.
//
// Key correction:
// - NOT_COMMUNICATING has TWO AND (orthogonal) substates/regions:
//   1) HOST-INITIATED CONNECT (WAIT_CR_FROM_HOST)
//   2) EQUIPMENT-INITIATED CONNECT (WAIT_CRA, WAIT_DELAY)
//
// Therefore, while enabled & not communicating, BOTH regions are active at once.
// Any region completing establishment can trigger transition to COMMUNICATING.
// (Table 3.2 rows #5 and #10 both apply on entry to NOT_COMMUNICATING.)
//
// This file is self-contained: external I/O is via Actions.
// Timer callback never mutates state directly; it emits an event.
//
// Package name can be changed to whatever you like.
package secs

import (
    "fmt"
    "sync"
    "time"
)

//
// ---------- Top-level enabled/disabled + communicating/not communicating ----------
//

type MajorState int

const (
    MajorDisabled MajorState = iota
    MajorEnabled
)

func (s MajorState) String() string {
    switch s {
    case MajorDisabled:
        return "DISABLED"
    case MajorEnabled:
        return "ENABLED"
    default:
        return fmt.Sprintf("MajorState(%d)", int(s))
    }
}

type EnabledState int

const (
    EnabledNotCommunicating EnabledState = iota
    EnabledCommunicating
)

func (s EnabledState) String() string {
    switch s {
    case EnabledNotCommunicating:
        return "NOT_COMMUNICATING"
    case EnabledCommunicating:
        return "COMMUNICATING"
    default:
        return fmt.Sprintf("EnabledState(%d)", int(s))
    }
}

//
// ---------- AND regions inside NOT_COMMUNICATING ----------
//

type HostRegionState int

const (
    HostWaitCRFromHost HostRegionState = iota // Table 3.2 #10/#15
)

func (s HostRegionState) String() string {
    switch s {
    case HostWaitCRFromHost:
        return "HOST_INITIATED_CONNECT/WAIT_CR_FROM_HOST"
    default:
        return fmt.Sprintf("HostRegionState(%d)", int(s))
    }
}

type EqpRegionState int

const (
    EqpWaitCRA EqpRegionState = iota // Table 3.2 #5/#6/#9
    EqpWaitDelay                     // Table 3.2 #6/#7/#8
)

func (s EqpRegionState) String() string {
    switch s {
    case EqpWaitCRA:
        return "EQP_INITIATED_CONNECT/WAIT_CRA"
    case EqpWaitDelay:
        return "EQP_INITIATED_CONNECT/WAIT_DELAY"
    default:
        return fmt.Sprintf("EqpRegionState(%d)", int(s))
    }
}

//
// ---------- Events (Triggers) ----------
//

type Event int

const (
    // Table 3.2 #1
    EvSystemInit Event = iota

    // Table 3.2 #2 / #3
    EvOperatorEnable
    EvOperatorDisable

    EvLinkDisconnected
    // Equipment region triggers (Table 3.2 #6/#7/#8)
    EvConnTransactionFail // connection transaction failure
    EvCommDelayExpired    // CommDelay timer expired

    // Messages
    EvRecvS1F13
    EvRecvOtherMsg
    EvRecvS9Fx

    // Table 3.2 #9 (equipment region completion)
    EvRecvExpectedS1F14_CommAck0

    // Table 3.2 #14
    EvCommFailure
)

func (e Event) String() string {
    switch e {
    case EvSystemInit:
        return "EvSystemInit"
    case EvOperatorEnable:
        return "EvOperatorEnable"
    case EvOperatorDisable:
        return "EvOperatorDisable"
    case EvLinkDisconnected:
        return "EvLinkDisconnected"
    case EvConnTransactionFail:
        return "EvConnTransactionFail"
    case EvCommDelayExpired:
        return "EvCommDelayExpired"
    case EvRecvS1F13:
        return "EvRecvS1F13"
    case EvRecvOtherMsg:
        return "EvRecvOtherMsg"
    case EvRecvS9Fx:
        return "EvRecvS9Fx"
    case EvRecvExpectedS1F14_CommAck0:
        return "EvRecvExpectedS1F14_CommAck0"
    case EvCommFailure:
        return "EvCommFailure"
    default:
        return fmt.Sprintf("Event(%d)", int(e))
    }
}

//
// ---------- Config ----------
//

type Config struct {
    // Table 3.2 #1: system default may be DISABLED or ENABLED.
    SystemDefault MajorState // MajorDisabled or MajorEnabled

    // Table 3.2: CommDelay timer used in EQP_INITIATED region.
    CommDelay time.Duration

    // Spec text: In NOT_COMMUNICATING, allow only S1F13/S1F14/S9Fx, discard others.
    StrictDiscard bool
}

func DefaultConfig() Config {
    return Config{
        SystemDefault:  MajorDisabled,
        CommDelay:     5 * time.Second,
        StrictDiscard: true,
    }
}

//
// ---------- Actions (external effects) ----------
//

type Actions interface {
    // Establish communications transactions
    SendS1F13()
    //SendS1F14_CommAck0()

    // Queue/spool handling
    DequeueAllMessagesQueuedToSend()

    // Discard inbound message (no reply)
    DiscardInbound(reason string)

    // Logging (optional but very useful for spec verification)
    Logf(format string, args ...any)
        TellUI()
}

//
// ---------- ComFSM ----------
//

type ComFSM struct {
    cfg Config
    a   Actions

    events chan Event

    mu      sync.Mutex
    major   MajorState
    enabled EnabledState

    // AND substates (valid only when enabled==NOT_COMMUNICATING)
    host HostRegionState
    eqp  EqpRegionState

    // CommDelay timer used by equipment region (WAIT_DELAY behavior)
    commDelayT *time.Timer
}

func CreateComFSM(cfg Config, a Actions) *ComFSM {
    m := &ComFSM{
        cfg:    cfg,
        a:      a,
        events: make(chan Event, 128),
        major:  MajorDisabled,
        // enabled/regions will be set on SystemInit when MajorEnabled
    }
    return m
}

// Emit injects an event into the FSM; safe from any goroutine.
func (m *ComFSM) Emit(ev Event) { m.events <- ev }

// TryEmit is a non-blocking variant.
func (m *ComFSM) TryEmit(ev Event) bool {
    select {
    case m.events <- ev:
        return true
    default:
        return false
    }
}

// Snapshot for logging/debugging.
type Snapshot struct {
    Major   MajorState
    Enabled EnabledState
    Host    HostRegionState
    Eqp     EqpRegionState
}

func (m *ComFSM) Snapshot() Snapshot {
    m.mu.Lock()
    defer m.mu.Unlock()
    return Snapshot{
        Major:   m.major,
        Enabled: m.enabled,
        Host:    m.host,
        Eqp:     m.eqp,
    }
}

func (m *ComFSM) Run() {
    select {
        case ev := <-m.events:
            m.handle(ev)
            m.a.TellUI()
        default:
            break
    }
}

//
// ---------- Timer helpers (Timer -> Event) ----------
//

func (m *ComFSM) startCommDelayTimerLocked() {
    m.stopCommDelayTimerLocked()

    d := m.cfg.CommDelay
    if d <= 0 {
        d = 1 * time.Millisecond
    }
    m.commDelayT = time.AfterFunc(d, func() {
        // IMPORTANT: do not mutate FSM state in timer callback
        m.Emit(EvCommDelayExpired)
    })
}

func (m *ComFSM) stopCommDelayTimerLocked() {
    if m.commDelayT != nil {
        m.commDelayT.Stop()
        m.commDelayT = nil
    }
}

func (m *ComFSM) stopTimers() {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.stopCommDelayTimerLocked()
}

//
// ---------- Transitions (Table 3.2 aligned) ----------
//

func (m *ComFSM) logStateLocked(prefix string) {
    m.a.Logf("[FSM] %s major=%s enabled=%s host=%s eqp=%s",
        prefix, m.major.String(), m.enabled.String(), m.host.String(), m.eqp.String())
}

// Entry to ENABLED leads to NOT_COMMUNICATING (Table 3.2 #4), and then
// entry actions for BOTH AND regions fire:
// - Equipment region entry (Table 3.2 #5) -> WAIT_CRA + init timer + send S1F13
// - Host region entry (Table 3.2 #10) -> WAIT_CR_FROM_HOST
func (m *ComFSM) enterEnabledNotCommunicatingLocked(why string) {
    m.major = MajorEnabled
    m.enabled = EnabledNotCommunicating

    // Enter AND regions simultaneously
    // Host region: Table 3.2 #10
    m.host = HostWaitCRFromHost

    // Equipment region: Table 3.2 #5
    m.eqp = EqpWaitCRA
    m.startCommDelayTimerLocked()
    m.a.SendS1F13()

    m.a.Logf("[FSM] ENTER ENABLED/NOT_COMMUNICATING (%s)", why)
    m.logStateLocked("after entry")
}

// Exit ENABLED -> DISABLED (Table 3.2 #3)
func (m *ComFSM) enterDisabledLocked(why string) {
    // Spec text: when switching to DISABLED, comms cease immediately; discard etc as needed.
    m.stopCommDelayTimerLocked()
    m.major = MajorDisabled
    // enabled/regions become irrelevant
    m.enabled = EnabledNotCommunicating
    m.host = HostWaitCRFromHost
    m.eqp = EqpWaitCRA

    m.a.Logf("[FSM] ENTER DISABLED (%s)", why)
}

func (m *ComFSM) enterCommunicatingLocked(why string) {
    // Leaving NOT_COMMUNICATING AND state -> COMMUNICATING.
    // Stop comm delay timer (equipment region) because comm is established.
    m.stopCommDelayTimerLocked()

    m.enabled = EnabledCommunicating
    m.a.Logf("[FSM] ENTER COMMUNICATING (%s)", why)
    m.logStateLocked("after entry")
}

func (m *ComFSM) reenterNotCommunicatingLocked(why string) {
    // From COMMUNICATING failure (Table 3.2 #14), return to NOT_COMMUNICATING and restart both regions.
    m.a.DequeueAllMessagesQueuedToSend()
    m.enterEnabledNotCommunicatingLocked(why)
}

//
// ---------- Event handler ----------
//

func (m *ComFSM) handle(ev Event) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.a.Logf("[FSM] event=%s", ev.String())
    m.logStateLocked("before")

    // --- Major DISABLED ---
    if m.major == MajorDisabled {
        switch ev {
        case EvSystemInit:
            // Table 3.2 #1: System initialization -> System Default
            if m.cfg.SystemDefault == MajorEnabled {
                // Table 3.2 #2/#4 implied entry to enabled
                m.enterEnabledNotCommunicatingLocked("Table3.2#1 SystemDefault=ENABLED")
            } else {
                m.enterDisabledLocked("Table3.2#1 SystemDefault=DISABLED")
            }

        case EvOperatorEnable:
            // Table 3.2 #2: DISABLED -> ENABLED (then entry to ENABLED => NOT_COMMUNICATING)
            m.enterEnabledNotCommunicatingLocked("Table3.2#2 Operator DISABLED->ENABLED")

        default:
            m.a.DiscardInbound("DISABLED: communications prohibited")
        }
        m.logStateLocked("after")
        return
    }

    // --- Major ENABLED common ---
    if ev == EvOperatorDisable {
        // Table 3.2 #3: ENABLED -> DISABLED from any enabled state
        m.enterDisabledLocked("Table3.2#3 Operator ENABLED->DISABLED")
        m.logStateLocked("after")
        return
    }

    // --- ENABLED substate handling ---
    switch m.enabled {

    case EnabledCommunicating:
        switch ev {
        case EvCommFailure:
            // Table 3.2 #14: COMMUNICATING + comm failure -> NOT_COMMUNICATING; Dequeue queued messages.
            m.reenterNotCommunicatingLocked("Table3.2#14 COMMUNICATING failure -> NOT_COMMUNICATING")

        case EvRecvS1F13:
            // Spec text: If receive S1F13 while COMMUNICATING, respond S1F14 COMMACK=0
            //m.a.SendS1F14_CommAck0()
        case EvLinkDisconnected:
            m.reenterNotCommunicatingLocked("Link down -> COMM failure")
        default:
            // Other COMMUNICATING messages handled by your SECS-II router (outside FSM)
        }

    case EnabledNotCommunicating:
        // Harel AND: host region + equipment region both active; same event can affect both.
        // Additionally enforce NOT_COMMUNICATING discard rule if configured.
        if m.cfg.StrictDiscard {
            switch ev {
            case EvRecvS1F13,  EvRecvS9Fx:
                // allowed
            case EvRecvOtherMsg:
                m.a.DiscardInbound("NOT_COMMUNICATING: discard non S1F13/S1F14/S9Fx")
                // Table 3.2 #8 says only WAIT_DELAY reacts specially to non-S1F13;
                // discarding here is fine; equipment region handler below may still decide to send S1F13.
            }
        }
        // ---- Host region handler ----
        hostTriggeredCommunicating := false
        switch m.host {
        case HostWaitCRFromHost:
            // Table 3.2 #15: WAIT_CR_FROM_HOST + recv S1F13 -> COMMUNICATING; Action: send S1F14 COMMACK=0
            if ev == EvRecvS1F13 {
                //m.a.SendS1F14_CommAck0()
                hostTriggeredCommunicating = true
            }
        }

        // ---- Equipment region handler ----
        eqpTriggeredCommunicating := false
        switch m.eqp {

        case EqpWaitCRA:
            switch ev {
            case EvConnTransactionFail:
                // Table 3.2 #6: WAIT_CRA + connection transaction failure -> WAIT_DELAY
                // Actions: init CommDelay timer; dequeue all queued-to-send
                m.startCommDelayTimerLocked()
                m.a.DequeueAllMessagesQueuedToSend()
                m.eqp = EqpWaitDelay

            case EvRecvExpectedS1F14_CommAck0:
                // Table 3.2 #9: WAIT_CRA + got expected S1F14 COMMACK=0 -> COMMUNICATING
                eqpTriggeredCommunicating = true
            case EvLinkDisconnected:
                 m.startCommDelayTimerLocked()
                 m.a.DequeueAllMessagesQueuedToSend()
                 m.eqp = EqpWaitDelay
            }

        case EqpWaitDelay:
            switch ev {
            case EvCommDelayExpired:
                // Table 3.2 #7: WAIT_DELAY expired -> WAIT_CRA; Action: send S1F13
                m.a.SendS1F13()
                m.eqp = EqpWaitCRA

            case EvRecvOtherMsg:
                // Table 3.2 #8: WAIT_DELAY + received msg other than S1F13 -> WAIT_CRA
                // Actions: discard, no reply; set timer expired; send S1F13
                // (We already discarded above if StrictDiscard, but discarding twice is harmless; keep spec clarity.)
                m.a.DiscardInbound("Table3.2#8 WAIT_DELAY got msg != S1F13; discard and attempt S1F13")
                m.a.SendS1F13()
                m.eqp = EqpWaitCRA
            }
        }

        // ---- Leaving AND state ----
        // If either region establishes communications, the whole NOT_COMMUNICATING(AND) exits to COMMUNICATING.
        // If both happen "simultaneously" due to same event, that's also fine.
        if hostTriggeredCommunicating {
            m.enterCommunicatingLocked("Table3.2#15 Host region established (recv S1F13)")
        } else if eqpTriggeredCommunicating {
            m.enterCommunicatingLocked("Table3.2#9 Equipment region established (recv expected S1F14 COMMACK=0)")
        } else {
            // No top-level state change; remain in NOT_COMMUNICATING with possibly updated region substates.
        }

    default:
        // unreachable
    }

    m.logStateLocked("after")
}
