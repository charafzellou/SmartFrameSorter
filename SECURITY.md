# Security policy

## Supported versions

Security fixes are provided for the latest published SmartFrameSorter release. Older releases may be assessed case by case, but users should upgrade to the latest verified bundle.

## Report a vulnerability

Please do not disclose a suspected vulnerability in a public issue, discussion, or pull request.

Use GitHub's [private vulnerability reporting form](https://github.com/charafzellou/SmartFrameSorter/security/advisories/new). Include:

- the affected SmartFrameSorter version or commit;
- the Windows and media environment used;
- clear reproduction steps or a minimal non-sensitive test case;
- the observed and expected behavior;
- the potential impact; and
- any suggested mitigation, if known.

Do not attach private, copyrighted, or personally identifying media. Create a minimal synthetic sample when a file is needed to demonstrate the issue.

The project will acknowledge a report when it is reviewed, investigate it, and coordinate disclosure and remediation based on severity and maintainer availability. Please allow a reasonable remediation period before public disclosure.

## Security scope

Relevant reports include unsafe file handling, output overwrite or rollback failures, command execution, malformed-media crashes with security impact, unexpected network activity, supply-chain or release-integrity problems, and vulnerabilities introduced by bundled dependencies.

Visual grouping quality problems without a security impact should be reported as regular bugs.

