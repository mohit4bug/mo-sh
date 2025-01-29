import {
  boolean,
  integer,
  jsonb,
  pgEnum,
  pgTable,
  text,
  timestamp
} from 'drizzle-orm/pg-core'

export const sourceType = pgEnum('source_type', ['github'])

export const sources = pgTable('sources', {
  id: text().primaryKey(),
  name: text().notNull(),
  type: sourceType().notNull(),
  createdAt: timestamp({ mode: 'string', withTimezone: true })
    .notNull()
    .defaultNow(),
  updatedAt: timestamp({ mode: 'string', withTimezone: true })
    .notNull()
    .defaultNow()
})

type Owner = {
  login: string
  id: number
  node_id: string
  avatar_url: string
  gravatar_id: string
  url: string
  html_url: string
  followers_url: string
  following_url: string
  gists_url: string
  starred_url: string
  subscriptions_url: string
  organizations_url: string
  repos_url: string
  events_url: string
  received_events_url: string
  type: string
  site_admin: boolean
}

type Permissions = {
  [key: string]: string
}

type Events = string[]

export const githubApps = pgTable('github_apps', {
  id: text().primaryKey(),
  slug: text().notNull(),
  clientId: text().notNull(),
  nodeId: text().notNull(),
  owner: jsonb().$type<Owner>().notNull(),
  name: text().notNull(),
  description: text().notNull(),
  externalUrl: text().notNull(),
  htmlUrl: text().notNull(),
  createdAt: timestamp({ mode: 'string', withTimezone: true }).notNull(),
  updatedAt: timestamp({ mode: 'string', withTimezone: true }).notNull(),
  permissions: jsonb().$type<Permissions>().notNull(),
  events: jsonb().$type<Events>().notNull(),
  sourceId: text()
    .notNull()
    .references(() => sources.id),
  clientSecret: text().notNull(),
  webhookSecret: text().notNull(),
  privateKeyId: text()
    .notNull()
    .references(() => privateKeys.id)
})

export const privateKeys = pgTable('private_keys', {
  id: text().primaryKey(),
  name: text().notNull(),
  key: text().notNull(),
  isExternal: boolean().notNull().default(true),
  createdAt: timestamp({ mode: 'string', withTimezone: true })
    .notNull()
    .defaultNow(),
  updatedAt: timestamp({ mode: 'string', withTimezone: true })
    .notNull()
    .defaultNow()
})

export const servers = pgTable('servers', {
  id: text().primaryKey(),
  name: text().notNull(),
  hostname: text().notNull(),
  port: integer().notNull(),
  privateKeyId: text()
    .notNull()
    .references(() => privateKeys.id),
  createdAt: timestamp({ mode: 'string', withTimezone: true })
    .notNull()
    .defaultNow(),
  updatedAt: timestamp({ mode: 'string', withTimezone: true })
    .notNull()
    .defaultNow()
})
