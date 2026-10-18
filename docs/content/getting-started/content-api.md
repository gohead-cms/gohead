# Content API

All content endpoints live under `/api` and require a bearer token obtained from `POST /auth/login`.

## Accounts and roles

GoHead ships with three roles:

| Role     | Content permissions              | Admin routes (`/admin/*`) |
|----------|----------------------------------|---------------------------|
| `admin`  | create, read, update, delete     | yes                       |
| `editor` | create, read, update, delete     | no                        |
| `viewer` | read                             | no                        |

`POST /auth/register` never lets the caller choose a role:

- the **first** account created on an empty instance becomes `admin`;
- every later self-registered account is a `viewer`.

Admins manage accounts and promote users through `/admin/users`:

```bash
# create a user with a role
curl -X POST $HOST/admin/users -H "Authorization: Bearer $ADMIN" \
  -d '{"username":"alice","email":"alice@example.com","password":"s3cretpass","role_name":"editor"}'

# promote / change password (only username, email, password and role_name are accepted)
curl -X PUT $HOST/admin/users/2 -H "Authorization: Bearer $ADMIN" -d '{"role_name":"editor"}'
```

Set `jwt_secret` (or `GOHEAD_JWT_SECRET`) to a long random value. If it is unset or left at the sample
value, the server generates a random secret at startup and issued tokens stop working after a restart.

## Listing items

`GET /api/collections/<collection>` supports pagination, filtering and sorting:

| Parameter                        | Meaning                                               |
|----------------------------------|-------------------------------------------------------|
| `page`, `pageSize`               | Pagination (`pageSize` between 1 and 100, default 10) |
| `filters[<field>][<op>]=<value>` | Filter on an attribute                                |
| `filters[<field>]=<value>`       | Shorthand for `$eq`                                   |
| `sort=<field>[:asc\|:desc],...`  | One or more sort keys                                 |

Operators: `$eq`, `$ne`, `$lt`, `$lte`, `$gt`, `$gte`, `$contains`.

`<field>` is any scalar attribute of the collection (`string`, `text`, `richtext`, `email`, `uid`,
`enum`, `int`, `float`, `decimal`, `bool`, `date`, `datetime`, `time`) or one of the built-in columns
`id`, `created_at`, `updated_at`. Numeric attributes compare numerically, `$contains` only applies to text,
and booleans only support `$eq`/`$ne`. Unknown or unsupported fields return `400`.

```bash
# published articles with at least 100 views, most viewed first
curl -G $HOST/api/collections/articles -H "Authorization: Bearer $TOKEN" \
  --data-urlencode 'filters[published]=true' \
  --data-urlencode 'filters[views][$gte]=100' \
  --data-urlencode 'sort=views:desc'
```

## Creating items

`POST /api/collections/<collection>` accepts either one item (`{"data": {...}}`) or an array of them.
A batch is atomic: if any item fails validation, nothing is stored and the response names the failing
`item_index`.

## GraphQL

`POST /api/graphql` accepts `query`, `variables` and `operationName`. Each collection gets a query field
and `create<Name>`, `update<Name>` and `delete<Name>` mutations. Updates are partial. GraphQL enforces the
same role permissions and validation as the REST API, and the schema is rebuilt automatically when a
collection is created, changed or deleted.

```graphql
mutation ($title: String!) {
  createarticles(input: { title: $title, views: 3 }) { id title views }
}
```

## Schema listings

Admins can list definitions with `GET /admin/collections`, `GET /admin/components` and
`GET /admin/singletons`.
