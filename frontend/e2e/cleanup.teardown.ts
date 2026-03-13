import { test as teardown } from "@playwright/test";
import fs from "fs";

teardown("remove auth state", async () => {
  if (fs.existsSync("e2e/.auth/user.json")) {
    fs.unlinkSync("e2e/.auth/user.json");
  }
});
