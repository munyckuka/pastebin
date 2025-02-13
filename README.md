# pastebin
Pastebin webservice on golang


**Pastebin** is a web service that allows users to save and share text snippets, often used for publishing code, error logs, or any textual data. The primary idea is to provide temporary or permanent storage for text-based information with options for public or private access.

![image](https://github.com/user-attachments/assets/3e78db94-49cc-4eca-8ca9-6a0fa538e2f1)


### Functionality
Here are the main features:

1. **Create a Paste** — The user enters text that is saved on the server.
2. **Edit a Paste** — Authorized users can edit or delete their pastes.
3. **Unique Link** — A unique link is generated to access the text.
4. **View a Paste** — Enables viewing pastes via the unique link.
5. **Time-to-Live (TTL)** — Allows setting a lifespan after which the paste is automatically deleted.
6. **Privacy** — Provides an option to choose between public and private pastes.

### How to start:
1. Open server/server.go and change link to DB var(clientOptions)
2. Start main.go
