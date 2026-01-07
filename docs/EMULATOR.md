# Emulator Support

The Asset Manager supports integration with the following Habbo emulators:

1.  **Arcturus Morningstar**
2.  **PlusEMU**
3.  **Comet**

## Configuration

To configure the target emulator, update your `.env` file or configuration parameters.

### Environment Variable

Set the `SERVER_EMULATOR` variable to one of the following values:

*   `arcturus` (Default)
*   `plusemu`
*   `comet`

Example `.env`:
```bash
SERVER_EMULATOR=arcturus
```

### Database Connection

The asset manager can optionally connect to the emulator's database. This connection is used to validate asset references against the emulator's items table.

Configuration in `.env`:
```bash
DATABASE_HOST=localhost
DATABASE_PORT=3306
DATABASE_USER=root
DATABASE_PASSWORD=yourpassword
DATABASE_NAME=emulator
```

The database connection is optional. If the connection fails, the server will log a warning but continue startup.
