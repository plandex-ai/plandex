---
sidebar_position: 9
sidebar_label: Collaboration / Orgs
---

# Collaboration and Orgs

While so far Plandex is mainly focused on a single-user experience, we plan to add features for sharing, collaboration, and team management in the future, and some groundwork has already been done. **Orgs** are the basis for collaboration in Plandex.

## Multiple Users

Orgs are helpful already if you have multiple users using Plandex in the same project. Because Plandex outputs a `.plandex` file containing a bit of non-sensitive config data in each directory a plan is created in, you'll have problems with multiple users unless you either get each user into the same org or put `.plandex` in your `.gitignore` file. Otherwise, each user will overwrite other users' `.plandex` files on every push, and no one will be happy.

## Domain Access

When starting out with Plandex and creating a new org, you have the option of automatically granting access to anyone with an email address on your domain.

## Invitations

If you choose not to grant access to your whole domain, or you want to invite someone from outside your email domain, you can use `plandex invite`:

```bash
plandex invite
```

## Joining an Org

To join an org you've been invited to, use `plandex sign-in`:

```bash
plandex sign-in
```

## Listing Users and Invites

To list users and pending invites, use `plandex users`:

```bash
plandex users
```

## Revoking Users and Invites

To revoke an invite or remove a user, use `plandex revoke`:

```bash
plandex revoke
```