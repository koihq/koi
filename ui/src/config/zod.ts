import { z } from "zod";

// Zod otherwise probes for `new Function`, which the Content Security Policy reports as a violation.
z.config({ jitless: true });
