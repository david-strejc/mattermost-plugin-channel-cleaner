# Mattermost Channel Cleaner Plugin

A Mattermost plugin that allows authorized users to clear all messages in a channel.

## Features

- Clear all messages in a channel with a single command
- Configurable permissions (Admin only, Channel Admin, or Team Admin)
- Built-in safety confirmation to prevent accidental deletions
- Convenient trash icon button in channel header
- Bot user for plugin operations

## Usage

### Slash Command
Type `/clearchannel` in any channel to initiate the clearing process. You'll be prompted to confirm with `/clearchannel confirm`.

### Channel Header Button
Click the trash icon in the channel header to clear messages. A confirmation dialog will appear before deletion.

## Installation

1. Download the latest release from the releases page
2. Upload the plugin through the Mattermost System Console > Plugins > Management
3. Enable the plugin

## Configuration

In the System Console > Plugins > Channel Cleaner:

- **Permission Level**: Choose who can clear channels:
  - Admin Only (default)
  - Channel Admin
  - Team Admin

## Building from Source

### Prerequisites
- Go 1.18+
- Node.js 14+

### Build Steps
```bash
# Clone the repository
git clone https://github.com/david-strejc/mattermost-plugin-channel-cleaner.git
cd mattermost-plugin-channel-cleaner

# Build server
cd server
go build -o dist/plugin-linux-amd64 .
cd ..

# Build webapp
cd webapp
npm install
npm run build
cd ..

# Package plugin
tar -czf mattermost-plugin-channel-cleaner.tar.gz plugin.json server/dist/plugin-linux-amd64 webapp/dist/ assets/
```

## Security

This plugin permanently deletes messages and cannot be undone. Ensure proper permissions are configured to prevent unauthorized use.

## License

MIT License