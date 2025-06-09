# Contributing to SimTool

Thank you for your interest in contributing to SimTool! This document provides guidelines for contributing to the project.

## Development Setup

1. **Prerequisites**
   - Go 1.21 or later
   - macOS with Xcode installed (for iOS simulator functionality)
   - Git

2. **Fork and Clone**
   ```bash
   git clone https://github.com/yourusername/simtool.git
   cd simtool
   ```

3. **Install Dependencies**
   ```bash
   go mod download
   ```

4. **Build and Run**
   ```bash
   make build
   ./simtool
   ```

## Code Style

- Follow standard Go conventions
- Run `gofmt` before committing: `make fmt`
- Ensure code passes linting: `golangci-lint run`
- Keep line length reasonable (around 100 characters)
- Add comments for exported functions and types

## Testing

- Write tests for new functionality
- Ensure all tests pass: `make test`
- Maintain or improve code coverage
- Test on different terminal emulators when possible

## Commit Messages

Follow the conventional commit format:

```
type(scope): subject

body (optional)

footer (optional)
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions or modifications
- `chore`: Build process or auxiliary tool changes

Example:
```
feat(tui): add search functionality to app list

Added ability to search apps by name, bundle ID, and version.
Includes keyboard shortcuts and visual feedback.

Closes #123
```

## Pull Request Process

1. **Create a Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Your Changes**
   - Follow the code style guidelines
   - Add tests for new functionality
   - Update documentation as needed

3. **Test Your Changes**
   ```bash
   make test
   make build
   ./simtool  # Manual testing
   ```

4. **Submit Pull Request**
   - Push your branch to your fork
   - Create a PR against the `main` branch
   - Fill in the PR template completely
   - Link any related issues

5. **PR Review Process**
   - PRs require at least one review
   - Address feedback promptly
   - Keep PRs focused and reasonably sized

## Development Guidelines

### Project Structure
```
simtool/
├── cmd/simtool/        # Application entry point
├── internal/           # Internal packages
│   ├── config/         # Configuration management
│   ├── simulator/      # iOS simulator interaction
│   ├── tui/           # Terminal UI implementation
│   └── ui/            # UI styling and utilities
```

### Key Principles
- Keep the UI responsive and fast
- Maintain cross-terminal compatibility
- Follow the Bubble Tea framework patterns
- Keep simulator operations efficient

### Adding New Features

1. Discuss major features in an issue first
2. Follow existing code patterns
3. Add configuration options if applicable
4. Update documentation
5. Add tests

## Questions?

Feel free to:
- Open an issue for questions
- Start a discussion
- Reach out to maintainers

Thank you for contributing!