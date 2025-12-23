# React + TypeScript + Vite

[![Coverage](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=coverage&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b) [![Duplicated Lines (%)](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=duplicated_lines_density&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b) [![Lines of Code](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=ncloc&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b) [![Quality Gate Status](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=alert_status&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b) [![Security Hotspots](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=security_hotspots&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b)

[![Reliability Issues](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=software_quality_reliability_issues&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b) [![Maintainability Issues](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=software_quality_maintainability_issues&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b) [![Security Issues](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=software_quality_security_issues&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b) [![Maintainability Rating](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=software_quality_maintainability_rating&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b) [![Reliability Rating](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=software_quality_reliability_rating&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b)

[![Security Rating](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=software_quality_security_rating&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b) [![Technical Debt](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=software_quality_maintainability_remediation_effort&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b)

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Babel](https://babeljs.io/) (or [oxc](https://oxc.rs) when used in [rolldown-vite](https://vite.dev/guide/rolldown)) for Fast Refresh
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/) for Fast Refresh

## React Compiler

The React Compiler is not enabled on this template because of its impact on dev & build performances. To add it, see [this documentation](https://react.dev/learn/react-compiler/installation).

## Expanding the ESLint configuration

If you are developing a production application, we recommend updating the configuration to enable type-aware lint rules:

```js
export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...

      // Remove tseslint.configs.recommended and replace with this
      tseslint.configs.recommendedTypeChecked,
      // Alternatively, use this for stricter rules
      tseslint.configs.strictTypeChecked,
      // Optionally, add this for stylistic rules
      tseslint.configs.stylisticTypeChecked,

      // Other configs...
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```

You can also install [eslint-plugin-react-x](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-x) and [eslint-plugin-react-dom](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-dom) for React-specific lint rules:

```js
// eslint.config.js
import reactX from 'eslint-plugin-react-x'
import reactDom from 'eslint-plugin-react-dom'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...
      // Enable lint rules for React
      reactX.configs['recommended-typescript'],
      // Enable lint rules for React DOM
      reactDom.configs.recommended,
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```
