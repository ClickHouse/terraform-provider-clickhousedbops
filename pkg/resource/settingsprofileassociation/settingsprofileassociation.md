You can use the `clickhousedbops_settings_profile_association` resource to associate a `settings_profile` to a `role` or `user` in a `ClickHouse` instance.

With `role_id` or `user_id`, the profile is set on the role or user with `ALTER ROLE` or `ALTER USER`.

With `user_name`, the user is added to the profile's `TO` list with `ALTER SETTINGS PROFILE` instead. Use it for users that `ALTER USER` cannot change, such as [JWT](https://clickhouse.com/docs/concepts/features/security/external-authenticators/jwt) ephemeral users. Their id is stable across tokens, so the assignment outlives the token.

ClickHouse only knows a JWT user while it holds a live token on that replica, which brings some limits:

- The user must exist when the association is created.
- While the user does not exist, it is missing from `system.settings_profiles.apply_to_list` and the provider keeps the association in state. Destroying it then leaves the profile applied to the user.
- ClickHouse has no `ADD TO`, so the provider rewrites the whole `TO` list. Entries it cannot name at that moment, such as other JWT users without a live token, are dropped. They come back as drift once those users log in again.
- A profile set `TO ALL` is refused.
