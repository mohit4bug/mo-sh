import { pgEnum, pgTable, text } from 'drizzle-orm/pg-core'

export const sourceType = pgEnum('source_type', ['github'])

export const sources = pgTable('sources', {
  id: text().primaryKey(),
  name: text().notNull(),
  type: sourceType().notNull()
})
