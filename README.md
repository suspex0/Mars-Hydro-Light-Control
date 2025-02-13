# Go MarsHydro LightControl

This project controls the brightness of a MarsHydro lamp based on a timer configuration. The application reads configuration files for timing and account credentials, calculates the expected brightness based on the time of day, and updates the lamp accordingly.

## Project Structure
- **main.go**: Initializes configurations, computes light timings, and runs the control loop.
- **internal/timer/timer.go**: Contains the logic for calculating expected brightness.
- **internal/config/config.go**: Handles loading configurations.
- **internal/api/mh_api.go**: Manages communications with the MarsHydro API.
- **timer.json**: Timer configuration file.
- **account.json**: Account configuration file.

## Configuration Setup

### Timer Configuration (timer.json)
Create or modify the file at:
```
/c:/Users/Story/source/code/Go_MarsHydro_LightControl/timer.json
```
Example configuration:
```json
{
  "StartHour": 8,
  "PlateauHour": 6,
  "PlateauOffset": -1,
  "Brightness": 100,
  "EndHour": 20,
  "StepSize": 10
}
```
- **StartHour**: Hour when the lamp starts to turn on.
- **PlateauHour**: Duration (in hours) during which the lamp remains at its maximum brightness.
- **PlateauOffset**: Offset (in hours) to adjust the sunrise/sunset ramp durations.
- **Brightness**: Maximum brightness during the plateau phase (i.e. the highest sun state).
- **EndHour**: Hour when the lamp turns off.
- **StepSize**: Brightness step increments (must be a multiple of 5).

### Account Configuration (account.json)
Create or modify the file at:
```
/c:/Users/Story/source/code/Go_MarsHydro_LightControl/account.json
```
Example configuration:
```json
{
    "Email": "email@provider.example",    
    "Password": "PasswordHere1234",
    "Wifiname": "Wifi1234", // Note: the Wi-Fi name is only used to make the API request seem legit. Providing your real Wi-Fi name is optional.
    "Timezone": "Europe/Berlin", 
    "Language": "German"
}
```
Place your actual credentials and network information here.


