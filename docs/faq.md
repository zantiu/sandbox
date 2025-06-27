# FAQ

**Q: Is this a server implementation?**  
A: No, this SDK provides helpers for building clients and server logic, but does not implement a server.

**Q: How do I generate models and clients from the OpenAPI spec?**  
A: Use the provided `generateNorthBound.sh` script in `sdk/api/wfm/`.

**Q: How do I add a new authentication method?**  
A: Implement the `Authenticator` interface in `sdk/auth/`.

**Q: Where do I find architecture documentation?**  
A: See [docs/design.md](./design.md).

**Q: How do I contribute?**  
A: See [docs/contributing.md](./contributing.md).