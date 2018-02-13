/**
 * Base configuration for non-development environments
 * (Production, test, staging, etc.)
 */
const eslintConfig = {
  "env": {
    "browser": true,
    "jest": true,
    "node": true,
    "es6": true,
    "mocha": true
  },
  "extends": [
    // Disable all unnecessary eslint rules, or those that may conflict with prettier
    "prettier",
  ],
  "parser": "babel-eslint",
  "plugins": ["jest", "prettier"],
  "parserOptions": {
    "ecmaVersion": 6,
    "sourceType": "module",
    "ecmaFeatures": {
      "arrowFunctions": true,
      "binaryLiterals": true,
      "blockBindings": true,
      "classes": true,
      "defaultParams": true,
      "destructuring": true,
      "forOf": true,
      "generators": true,
      "objectLiteralComputedProperties": true,
      "objectLiteralDuplicateProperties": true,
      "objectLiteralShorthandMethods": true,
      "objectLiteralShorthandProperties": true,
      "octalLiterals": true,
      "regexUFlag": true,
      "regexYFlag": true,
      "spread": true,
      "superInFunctions": true,
      "templateStrings": true,
      "unicodeCodePointEscapes": true,
      "globalReturn": true,
      "experimentalObjectRestSpread": true,
    },
  },

  "rules": {
    // The second parameter to the Prettier plugin is an array of options;
    // see https://github.com/prettier/eslint-plugin-prettier#options.
    //
    // The third option specifies the format of the pragma that triggers a Prettier pass.
    // In other words, only files prefixed with `/** @prettier */` will be checked.
    // Note that the pragma MUST be the very first comment in the file, as far as I can tell.
    // (Please use as a one-line comment rather than a block; eventually _all_ files should have
    // prettier enabled by default, and a one-line comment is easier to search/replace.)
    "prettier/prettier": ["error", {
      semi: false,
      singleQuote: true,
      trailingComma: 'all',
    }, "@prettier"],

    //
    // Possible Errors
    //
    // The following rules point out areas where you might have made mistakes.
    //
    "comma-dangle": ["error", "always-multiline"], // for nicer diffs :-)
    "no-cond-assign": "error", // disallow assignment in conditional expressions
    "no-console": ["error", {"allow": ["warn", "error"]}], // disallow use of console (off by default in the node environment)
    "no-constant-condition": ["error", {"checkLoops": false}], // disallow use of constant expressions in conditions
    "no-control-regex": "error", // disallow control characters in regular expressions
    "no-debugger": "error", // disallow use of debugger
    "no-dupe-args": "error", // disallow duplicate arguments in functions
    "no-dupe-keys": "error", // disallow duplicate keys when creating object literals
    "no-duplicate-case": "error", // disallow a duplicate case label.
    "no-empty": "error", // disallow empty statements
    "no-empty-character-class": "error", // disallow the use of empty character classes in regular expressions
    "no-ex-assign": 2, // disallow assigning to the exception in a catch block
    "no-extra-boolean-cast": 2, // disallow double-negation boolean casts in a boolean context
    "no-extra-parens": 0, // disallow unnecessary parentheses (off by default)
    "no-extra-semi": 2, // disallow unnecessary semicolons
    "no-func-assign": 2, // disallow overwriting functions written as function declarations
    "no-inner-declarations": 2, // disallow function or variable declarations in nested blocks
    "no-invalid-regexp": 2, // disallow invalid regular expression strings in the RegExp constructor
    "no-irregular-whitespace": 2, // disallow irregular whitespace outside of strings and comments
    "no-negated-in-lhs": 2, // disallow negation of the left operand of an in expression
    "no-obj-calls": 2, // disallow the use of object properties of the global object (Math and JSON) as functions
    "no-regex-spaces": 2, // disallow multiple spaces in a regular expression literal
    "quote-props": 2, // disallow reserved words being used as object literal keys (off by default)
    "no-sparse-arrays": 2, // disallow sparse arrays
    "no-unreachable": 2, // disallow unreachable statements after a return, throw, continue, or break statement
    "use-isnan": 2, // disallow comparisons with the value NaN
    "valid-jsdoc": 2, // Ensure JSDoc comments are valid (off by default)
    "valid-typeof": 2, // Ensure that the results of typeof are compared against a valid string

    //
    // Best Practices
    //
    // These are rules designed to prevent you from making mistakes.
    // They either prescribe a better way of doing something or help you avoid footguns.
    //
    "block-scoped-var": 0, // treat var statements as if they were block scoped (off by default). 0: deep destructuring is not compatible https://github.com/eslint/eslint/issues/1863
    "complexity": 0, // specify the maximum cyclomatic complexity allowed in a program (off by default)
    "consistent-return": 2, // require return statements to either always or never specify values
    "curly": 2, // specify curly brace conventions for all control statements
    "default-case": ["error"], // require default case in switch statements (off by default)
    "dot-notation": 2, // encourages use of dot notation whenever possible
    "eqeqeq": 2, // require the use of === and !==
    "guard-for-in": 2, // make sure for-in loops have an if statement (off by default)
    "no-alert": 2, // disallow the use of alert, confirm, and prompt
    "no-caller": 2, // disallow use of arguments.caller or arguments.callee
    "no-div-regex": 2, // disallow division operators explicitly at beginning of regular expression (off by default)
    "no-else-return": 2, // disallow else after a return in an if (off by default)
    "no-labels": 2, // disallow use of labels for anything other than loops and switches
    "no-eq-null": 2, // disallow comparisons to null without a type-checking operator (off by default)
    "no-eval": 2, // disallow use of eval()
    "no-extend-native": 2, // disallow adding to native types
    "no-extra-bind": 2, // disallow unnecessary function binding
    "no-fallthrough": 2, // disallow fallthrough of case statements
    "no-floating-decimal": 2, // disallow the use of leading or trailing decimal points in numeric literals (off by default)
    "no-implied-eval": 2, // disallow use of eval()-like methods
    "no-iterator": 2, // disallow usage of __iterator__ property
    "no-labels": 2, // disallow use of labeled statements
    "no-lone-blocks": 2, // disallow unnecessary nested blocks
    "no-loop-func": 2, // disallow creation of functions within loops
    "no-multi-spaces": ["error", {exceptions: {"VariableDeclarator": true}}],// disallow use of multiple spaces
    "no-multi-str": 2, // disallow use of multiline strings
    "no-native-reassign": 2, // disallow reassignments of native objects
    "no-new": 2, // disallow use of new operator when not part of the assignment or comparison
    "no-new-func": 2, // disallow use of new operator for Function object
    "no-new-wrappers": 2, // disallows creating new instances of String,Number, and Boolean
    "no-octal": 2, // disallow use of octal literals
    "no-octal-escape": 2, // disallow use of octal escape sequences in string literals, such as var foo = "Copyright \251";
    "no-param-reassign": 2, // disallow reassignment of function parameters (off by default)
    "no-process-env": 2, // disallow use of process.env (off by default)
    "no-proto": 2, // disallow usage of __proto__ property
    "no-redeclare": 2, // disallow declaring the same variable more then once
    "no-return-assign": 2, // disallow use of assignment in return statement
    "no-script-url": 2, // disallow use of javascript: urls.
    "no-self-compare": 2, // disallow comparisons where both sides are exactly the same (off by default)
    "no-sequences": 2, // disallow use of comma operator
    "no-throw-literal": 2, // restrict what can be thrown as an exception (off by default)
    "no-unused-expressions": 2, // disallow usage of expressions in statement position
    "no-void": 2, // disallow use of void operator (off by default)
    "no-warning-comments": [0, {"terms": ["todo", "fixme"], "location": "start"}], // disallow usage of configurable warning terms in comments": 2, // e.g. TODO or FIXME (off by default)
    "no-with": 2, // disallow use of the with statement
    "vars-on-top": 2, // requires to declare all vars on top of their containing scope (off by default)
    "wrap-iife": 2, // require immediate function invocation to be wrapped in parentheses (off by default)
    "yoda": 2, // require or disallow Yoda conditions

    //
    // Strict Mode
    //
    // These rules relate to using strict mode.
    //
    "strict": 0, // controls location of Use Strict Directives. 0: required by `babel-eslint`

    //
    // Variables
    //
    // These rules have to do with variable declarations.
    //
    "no-catch-shadow": 2, // disallow the catch clause parameter name being the same as a variable in the outer scope (off by default in the node environment)
    "no-delete-var": 2, // disallow deletion of variables
    "no-label-var": 2, // disallow labels that share a name with a variable
    "no-shadow": 2, // disallow declaration of variables already declared in the outer scope
    "no-shadow-restricted-names": 2, // disallow shadowing of names such as arguments
    "no-undef": 2, // disallow use of undeclared variables unless mentioned in a /*global */ block
    "no-undef-init": 2, // disallow use of undefined when initializing variables
    "no-undefined": 2, // disallow use of undefined variable (off by default)
    "no-unused-vars": ["error", {"args": "none"}], // disallow declaration of variables that are not used in the code
    "no-use-before-define": 0, // disallow use of variables before they are defined

    //
    // Stylistic Issues
    //
    // These rules are purely matters of style and are quite subjective.
    //
    "indent": ["error", 2, {"SwitchCase":1}], // this option sets a specific tab width for your code (off by default)
    "brace-style": 1, // enforce one true brace style (off by default)
    "camelcase": 1, // require camel case names
    "comma-spacing": [1, {"before": false, "after": true}], // enforce spacing before and after comma
    "comma-style": [1, "last"], // enforce one true comma style (off by default)
    "consistent-this": [1, "_this"], // enforces consistent naming when capturing the current execution context (off by default)
    "eol-last": 1, // enforce newline at the end of file, with no multiple empty lines
    "func-names": 0, // require function expressions to have a name (off by default)
    "func-style": 0, // enforces use of function declarations or expressions (off by default)
    "key-spacing": [1, {"beforeColon": false, "afterColon": true}], // enforces spacing between keys and values in object literal properties
    "new-cap": [1, {newIsCap: true, capIsNew: false}], // require a capital letter for constructors
    "new-parens": 1, // disallow the omission of parentheses when invoking a constructor with no arguments
    "newline-after-var": 0, // allow/disallow an empty newline after var statement (off by default)
    "no-array-constructor": 1, // disallow use of the Array constructor
    "no-inline-comments": 1, // disallow comments inline after code (off by default)
    "no-lonely-if": 1, // disallow if as the only statement in an else block (off by default)
    "no-mixed-spaces-and-tabs": 1, // disallow mixed spaces and tabs for indentation
    "no-multiple-empty-lines": [1, {"max": 2}], // disallow multiple empty lines (off by default)
    "no-nested-ternary": 1, // disallow nested ternary expressions (off by default)
    "no-new-object": 1, // disallow use of the Object constructor
    "no-spaced-func": 1, // disallow space between function identifier and application
    "no-ternary": 0, // disallow the use of ternary operators (off by default)
    "no-trailing-spaces": 1, // disallow trailing whitespace at the end of lines
    "no-extra-parens": ["warn", "all", {"returnAssign": false}], // disallow wrapping of non-IIFE statements in parens
    "one-var": [1, "never"], // allow just one var statement per function (off by default)
    "operator-assignment": [1, "never"], // require assignment operator shorthand where possible or prohibit it entirely (off by default)
    "padded-blocks": [1, "never"], // enforce padding within blocks (off by default)
    "quote-props": [1, "as-needed"], // require quotes around object literal property names (off by default)
    "quotes": [2, "single", {"avoidEscape": true, "allowTemplateLiterals": true}], // specify whether double or single quotes should be used
    "semi": ["off", "never"], // require or disallow use of semicolons instead of ASI
    "semi-spacing": [1, {"before": false, "after": true}], // enforce spacing before and after semicolons
    "sort-vars": 0, // sort variables within the same declaration block (off by default)
    "keyword-spacing": [1, {"before": true, "after": true}], // require a space after certain keywords (off by default)
    "space-before-blocks": [1, "always"], // require or disallow space before blocks (off by default)
    "space-before-function-paren": [1, {"anonymous": "always", "named": "never"}], // require or disallow space before function opening parenthesis (off by default)
    "space-in-parens": [1, "never"], // require or disallow spaces inside parentheses (off by default)
    "space-unary-ops": [1, {"words": true, "nonwords": false}], // Require or disallow spaces before/after unary operators (words on by default, nonwords off by default)
    "spaced-comment": ["error", "always"], // require or disallow a space immediately following the // in a line comment (off by default)
    "wrap-regex": 0, // require regex literals to be wrapped in parentheses (off by default)

    //
    // ECMAScript 6
    //
    // These rules are only relevant to ES6 environments and are off by default.
    //
    "no-var": "error", // require let or const instead of var (off by default)
    "generator-star-spacing": ["error", "before"], // enforce the spacing around the * in generator functions (off by default)

    //
    // Legacy
    //
    // The following rules are included for compatibility with JSHint and JSLint.
    // While the names of the rules may not match up with the JSHint/JSLint counterpart,
    // the functionality is the same.
    //
    "max-depth": ["error", 3], // specify the maximum depth that blocks can be nested (off by default)
    "max-len": ["error", 100, 2], // specify the maximum length of a line in your program (off by default)
    "max-params": ["error", 6], // limits the number of parameters that can be used in the function declaration. (off by default)
  }
}

/**
 * "Relaxed" linting mode B-)
 * (This is primarily for local development.)
 */
if (!!process.env.TBN_LINT_DEV) {
    eslintConfig.rules["no-console"] = ["warn", { allow: ["warn", "error"]}]
}

module.exports = eslintConfig
