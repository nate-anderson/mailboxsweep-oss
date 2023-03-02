import { Mailboxsweep } from "./client-gen";

export const API = new Mailboxsweep({
  BASE: "https://localhost:8081",
  WITH_CREDENTIALS: true,
}).default;
