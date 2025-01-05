# Contributing to Gohead

Thank you for considering contributing to **Gohead**! Your contributions are highly valued and help improve the project for everyone. This document outlines the process to get started.


## How Can You Contribute?

- Reporting bugs
- Suggesting new features
- Improving documentation
- Submitting code (fixes, features, or improvements)
- Writing tests

## Getting Started

1. **Fork the Repository**:
   - Click the "Fork" button on the repository page: [Gohead GitLab Repository](https://gitlab.com/sudo.bngz/gohead).

2. **Clone Your Fork**:
   ```bash
   git clone https://gitlab.com/your-username/gohead.git
   cd gohead
   ```

3. **Set Upstream**:
   ```bash
   git remote add upstream https://gitlab.com/sudo.bngz/gohead.git
   ```

4. **Create a New Branch**:
   Always create a new branch for your contributions:
   ```bash
   git checkout -b feature/your-feature-name
   ```

---

## Code Style

- Follow the **Golang standard style guidelines**.
- Use `gofmt` or `gocritic` to format your code before submitting.
- Maintain consistency with the existing codebase.

## Commit Messages

Use this format for commit messages:  
`<type>(<scope>): <description>`

### Examples:
- `feat(api): add endpoint for user authentication`
- `fix(ui): resolve alignment issue in header`
- `docs(readme): update contributing section`

### Allowed Types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code (white-space, formatting, etc.)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding or fixing tests
- `chore`: Changes to the build process or auxiliary tools

## Merge Request Guidelines

1. Ensure your code is **up to date** with the main branch:
   ```bash
   git pull upstream main
   ```

2. **Run Tests**:
   Make sure your code passes all tests before submitting:
   ```bash
   go test ./...
   ```

3. **Submit a Merge Request**:
   - Push your branch to your fork:
     ```bash
     git push origin feature/your-feature-name
     ```
   - Open a merge request (MR) on the original repository: [Gohead GitLab Repository](https://gitlab.com/sudo.bngz/gohead).

4. **Describe Your Changes**:
   In the merge request description, explain:
   - What problem your changes solve
   - How they solve it
   - Any additional context or considerations

---

## Reporting Issues

- Before reporting a bug, check if it has already been reported in the [Issues](https://gitlab.com/sudo.bngz/gohead/-/issues).
- Include detailed steps to reproduce the issue.
- Provide information about your environment:
  - Go version
  - Operating system

---

## Need Help?

Feel free to reach out by:
- Opening an issue in the [Issues section](https://gitlab.com/sudo.bngz/gohead/-/issues).
- Contacting maintainers via GitLab.

Thank you for contributing to Gohead! 🚀