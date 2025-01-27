import { type Config } from 'drizzle-kit'

export default {
  schema: 'src/db/schema.ts',
  dialect: 'postgresql',
  dbCredentials: {
    url: 'postgres://user:password@localhost:5432/db'
  },
  out: 'drizzle',
  casing: 'snake_case'
} satisfies Config
