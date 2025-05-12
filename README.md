# waybar-printer-status
A simple widget for displaying the Bambu Lab printer status in waybar.

I've tested the widget only with Bambu Lab X1C. There are some differences in the MQTT between Bambu Lab printer models, so the software might not work with other models.

The widget can be used with multiple monitors. Only one instance of the program will handle the MQTT messages; passing it to other instances by unix socket at `/tmp/waybar-bambu-status.sock`.

# Configuration
`$HOME/.config/waybar-bambu-status/config.json`
## Example
```json
{
	"printer": {
		"access_code": "",
		"serial": "",
		"address": "ssl://:8883",
		"mqtt_topic": "#",
		"username": "bblp",
		"client_id": "waybar-bambu-status"
	}
}
```

# Waybar configuration
## Add module to waybar
```
    "modules-right": [
        "custom/bambu_status",
    ],
```
## `bambu_status`-module configuration
```json
    "custom/bambu_status" : {
        "exec": "$HOME/bin/waybar_bambu_status",
        "return-type": "json"
    }

```

# Styling
Three classes are used in the styling: `running`, `idle` and `fault` 
```css
#custom-bambu_status.running {
    color: @green;
}

#custom-bambu_status.idle {
    color: alpha(@teal, 0.4);
}

#custom-bambu_status.fault {
    color: @red ;
}
```
