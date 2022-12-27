import {Mailboxsweep} from "./client-gen"

export const API = new Mailboxsweep({
    BASE: "http://localhost:8080"
}).default;