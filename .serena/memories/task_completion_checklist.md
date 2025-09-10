# Task Completion Checklist for Mailpit

When completing any development task in Mailpit, ensure you:

## Before Committing

### For Go Code Changes
1. **Format the code**: Run `gofmt -s -w .` to format all Go files
2. **Verify formatting**: Run `gofmt -s -d .` and ensure no output (no differences)
3. **Run tests**: Execute relevant tests
   - For specific packages: `go test ./internal/storage ./server -v`
   - For all tests: `go test -p 1 ./internal/storage ./server ./internal/smtpd ./internal/pop3 ./internal/tools ./internal/html2text ./internal/htmlcheck ./internal/linkcheck -v`
4. **Check for compilation**: Run `go build` to ensure the project builds

### For Frontend Changes
1. **Lint JavaScript/Vue**: Run `npm run lint`
2. **Fix linting issues**: Run `npm run lint-fix` if needed
3. **Build assets**: Run `npm run build` or `npm run package`
4. **Test the build**: Ensure the frontend builds without errors

### For Both Frontend and Backend Changes
1. Complete all Go checks above
2. Complete all Frontend checks above
3. Build full application:
   ```bash
   npm run package
   CGO_ENABLED=0 go build -ldflags "-s -w" -o mailpit
   ```
4. Test the built binary: `./mailpit`

## Before Opening a PR
1. Ensure all tests pass
2. Ensure all linting passes
3. Update relevant documentation if needed
4. Target the `develop` branch for PRs
5. Write clear commit messages
6. Fill out the PR template completely

## CI/CD Validation
The GitHub Actions workflow will automatically:
- Run Go tests on multiple OS (Ubuntu, Windows, macOS)
- Check Go formatting
- Run JavaScript linting
- Build and package the frontend
- Validate OpenAPI definitions
- Run benchmarks

Make sure your changes will pass these checks before pushing.