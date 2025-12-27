## Requirements

### System Requirements

- Linux environment
- Go (Golang)

### Go Installation

1. Download the Go SDK from:
   https://go.dev/dl/

2. Extract the archive and move the `go` directory to `/var`:

   tar -xzf go*.tar.gz  
   sudo mv go /var/
---

## Build

From the project root directory:

cd secs
make all
sudo cp resource /srv/secs -r

After a successful build, **two binaries** will be generated:

- **Equipment Server**  
  src/webserver/main

- **Host Server**  
  src/webHost/main

---

## Run

Start both servers in separate terminals (or run them in the background):

./src/webserver/main  
./src/webHost/main  

Each binary launches its own HTTP web server:

- Equipment server listens on **port 8080**
- Host server listens on **port 8090**

---

## Access via Browser

Open a web browser and navigate to:

- **Equipment Web UI, It is default passive mode so start it first **
  http://yourip:8080/site/equipment.html

- **Host Web UI**
  http://yourip:8090/site/host.html

Note: port is different  

--
## Dev note

| Requirement                        | Section References | Description                                                                               |
| ---------------------------------- | ------------------ | -----------------------------------------------------------------------------------       |
| State Models                       | 3.0, 3.1, 3.3      | (control state : S1F0/S1F1/S1F2/S1F15/S1F16/S1F17/S1F18) <BR> (HSMS-SS : control message) |
| Equipment Processing States        | 3.4                | (x)                                                                                       |
| Host-Initiated S1,F13/F14 Scenario | 4.1.5.1            | (S1F13(V)/S1F14(V))                                                                       |
| Event Notification                 | 4.2.1.1            | (S6F11/S6F12/S6F15/S6F16)                                                                 |
| On-line Identification             | 4.2.6              | (S1F1/S1F2)                                                                               |
| Error Messages                     | 4.9                | (S9F1/S9F3/S9F5/S9F7/S9F9)                                                                |
| Control (Operator-Initiated)       | 4.12               | (S1F0/S1F1/S1F2/S6F11/S6F12)                                                              |
| Documentation                      | 8.4                | (X)                                                                                       |



| Capability                         | Section References | Description                                             |
| ---------------------------------- | ------------------ | ------------------------------------------------------- |
| Establish Communications           | 4.1, 3.2           | (Communications State Model | S1F13(V)/S1F14(V))        |
| Event Notification                 | 4.2.1.1            | (S6F11/S6F12/S6F15/S6F16)                               |
| Dynamic Event Report Configuration | 4.2.1.2            | (S2F33/S2F34/S2F35/S2F36/S2F37/S2F38)                   |
| Variable Data Collection           | 4.2.2              | (S6F19)                                                 |
| Trace Data Collection              | 4.2.3              | (S2F23/S2F24/S6F1/S6F2)                                 |
| Limits Monitoring                  | 4.2.4              |                                                         |
| Status Data Collection             | 4.2.5              | (S1F3/S1F4/S1F11/S1F12)                                 |
| On-line Identification             | 4.2.6              | (S1F1/S1F2)                                             |
| Alarm Management                   | 4.3                | (S5F1/S5F2/S5F3/S5F4/S5F5/S5F6)                         |
| Remote Control                     | 4.4                | (S2F41/S2F42/S2F49(X)/S2F50(X))                         |
| Equipment Constants                | 4.5                | (S2F13/S2F14/S2F15/S2F16/S2F29/S2F30)                   |
| Process Program Management         | 4.6                |                                                         |
| Material Movement                  | 4.7                |                                                         |
| Equipment Terminal Services        | 4.8                | (S10F1/S10F2/S10F3/S10F4/S6F11/S6F12)                   |
| Error Messages                     | 4.9                | (S9F1/S9F3/S9F5/S9F7/S9F9)                              |
| Clock                              | 4.10               | (S2F17(X)/S2F18(X)/S2F31(X)/S2F32(X) NTP is preferred ) |
| Spooling                           | 4.11               |                                                         |
| Control (Operator-Initiated)       | 4.12               | (S1F0/S1F1/S1F2/S6F11/S6F12)                            |
| Control (Host-Initiated)           | 4.12.5.1           | (S1F15/S1F16/S1F17/S1F18/S6F11/S6F12)                   |



S1F0  SEND by qruipment(not format check)  
S1F1  formar checked | SEND by both  
S1F2  formar checked | SEND by both  
S1F3  format checked | SEND by HOST  
S1F4                 | SEND by EQUIPMENT  
S1F11 format checked | SEND by HOST  
S1F12                  | SEND by EQUIPMENT  
S1F13 format checked | SEND by both  
S1F14 format checked | SEND by both  
S1F15 format checked | SEND by HOST  
S1F16                | SEND by EQUIPMENT  
S1F17 format checked | SEND by HOST  
S1F18                | SEND by EQUIPMENT  

S2F13 format checked | SEND by HOST  
S2F14                | SEND by EQUIPMENT  
S2F15 format checked | SEND by HOST  
S2F16                | SEND by EQUIPMENT  
S2F23 format checked | SEND by HOST  
S2F24                | SEND by EQUIPMENT  
S2F29 format checked | SEND by HOST  
S2F30                | SEND by EQUIPMENT  
S2F33 format checked | SEND by HOST  
S2F34                | SEND by EQUIPMENT  
S2F35 format checked | SEND by HOST  
S2F36                | SEND by EQUIPMENT  
S2F37 format checked | SEND by HOST  
S2F38                | SEND by EQUIPMENT  
S2F41 format checked | SEND by HOST  
S2F42                | SEND by EQUIPMENT  

S5F1                 | SEND by EQUIPMENT  
S5F2  format checked | SEND by HOST  
S5F3  format checked | SEND by HOST  
S5F4                 | SEND by EQUIPMENT  
S5F5  format checked | SEND by HOST  
S5F6                 | SEND by EQUIPMENT  

S6F1                 | SEND by EQUIPMENT  
S6F2 format checked  | SEND by HOST  
S6F11                | SEND by EQUIPMENT  
S6F12 format checked | SEND by HOST  
S6F15 format checked | SEND by HOST  
S6F16                | SEND by EQUIPMENT  
S6F19 format checked | SEND by HOST  
S6F20                | SEND by EQUIPMENT  

S9F1                 | SEND by EQUIPMENT  
S9F3                 | SEND by EQUIPMENT  
S9F5                 | SEND by EQUIPMENT  
S9F7                 | SEND by EQUIPMENT  
S9F9                 | SEND by EQUIPMENT  

S10F1                 | SEND by EQUIPMENT  
S10F2 format checked  | SEND by HOST  
S10F3 format checked  | SEND by HOST  
S10F4                 | SEND by EQUIPMENT  
