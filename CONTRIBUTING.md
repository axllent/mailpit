# Contributing guide

Thank you for your interest in contributing to Mailpit, your help is greatly appreciated! Please follow the guidelines below to ensure a smooth contribution process.


## Code of conduct

Please be respectful and considerate in all interactions. Mailpit is open source and free of charge, however is the result of thousands of hours of work.


## Reporting issues and feature requests

If you find a bug or have a feature request, please [open an issue](https://github.com/axllent/mailpit/issues) and provide as much detail as possible. Pleas do not report security issues here (see below).


## Reporting security issues

Please do not report security issues publicly in GitHub. Refer to [SECURITY document](https://github.com/axllent/mailpit/blob/develop/.github/SECURITY.md) for instructions and contact information.



## How to contribute (pull request)

1. **Fork the repository**  
   Click the "Fork" button at the top right of this repository to create your own copy.

2. **Clone your fork**  
   ```bash
   git clone https://github.com/your-username/mailpit.git
   cd mailpit
   ```

3. **Create a branch**  
   Use a descriptive branch name:
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make your changes**  
   Write clear, concise code and include comments where necessary.

5. **Test your changes**  
   Run all tests to ensure nothing is broken. This is a mandatory step as pull requests cannot be merged unless they pass the automated testing.

6. **Ensure your changes pass linting**  
   Ensure your changes pass the [code linting](https://mailpit.axllent.org/docs/development/code-linting/) requirements. This is a mandatory step as pull requests cannot be merged unless they pass the automated linting tests.

7. **Commit and push**  
   Write a clear commit message:
   ```bash
   git add .
   git commit -m "Describe your changes"
   git push origin feature/your-feature-name
   ```

8. **Open a pull request**  
   Go to your fork on GitHub and open a pull request against the `develop` branch. Fill out the PR template and describe your changes.

---

Thank you for helping make this project awesome!
