# Nomos Technical Challenge

### Business context

[EDIFACT](https://en.wikipedia.org/wiki/EDIFACT) messages are the backbone of all external communication for electricity retailers in Germany. Via this communication channel, we receive all legally relevant information with grid operators, metering point operators, and other market partners (e.g. consumption data, tariff information, and billing-relevant updates). Every process step in the customer lifecycle (signup, invoicing, termination, …) depends on sending or receiving the correct EDIFACT messages, which are following strict formats and timelines defined by a central authority.

Because these exchanges are mandatory and fully standardized, a retailer must process huge volumes of EDIFACT messages reliably, validate them, and handle exceptions or errors. In practice, this means building robust automation around parsing, routing, reconciling, and responding to these messages.

### EDIFACT File Structure

- File sizes range between a few kilo bytes and up to a few mega bytes
- There are different message types for different use cases (e.g. MSCONS = sending meter reading information, INVOIC = sending an invoice for the grid usage, UTILMD = metadata changes of a customer, …)
- Plain-text format with line-like “segments,” each on a single line ending with `'`. Each segment consists of data elements separated by `+` and, if nested, by `:`.
- Files start with an interchange header (`UNB`) and end with an interchange trailer (`UNZ`). Each contained message is wrapped in its own message header (`UNH`) and message trailer (`UNT`).
- See an example in the [*Additional Resources*](https://www.notion.so/Nomos-Technical-Challenge-2f8901a06cac809187f3df887bacbb1b?pvs=21) section

### Additional complexities

- For some message types data batching is allowed across multiple customers. This means a single file can contain multiple messages assigned to multiple subscriptions.
- We are building while doing. So it must be possible that we re-process the history to extract a new data point where we learned we need it.
- Twice a year the format/ruleset of the EDIFACT files changes, sometime more and sometimes less. This is a hard switch on all market participants (with some shutting of their system)

---

# Your task

Design and implement a **minimal but realistic EDIFACT ingestion system** for **incoming messages only**.

The goal is not to perfectly model the German energy market, but to demonstrate how you would approach building a production-grade data processing pipeline for this kind of problem.

You are free to choose:

- Programming language and frameworks
- Storage technology (database, object storage, etc.)
- Use your Coding Agent of choice

### Functional Requirements

Your system should:

1. Accept EDIFACT files as input via API
2. Parse the file into individual messages. We’ve included 15 files as reference:

   [test_files.zip](attachment:094b28d1-41f0-4edc-a54a-8aa79b929779:test_files.zip)

3. Extract and persist at least:
    - Message type (e.g. MSCONS, INVOIC, …)
    - Interchange data from `UNB`
    - Message ID (from `UNH`)
    - Link to a subscription
    - Any other metadata you find worth to save.
4. Store the **raw file** as well as the **parsed/structured representation**.
5. Be able to reprocess previously ingested files.

We do **not** want you to implement a perfect EDIFACT parser for all message formats. Instead, we are keen to know what parsing strategy you would chose and how that would fit into the overall design.

Include a solution.md and briefly outline your design, key decision, assumptions & trade offs of your solution. Also outline how you would adjust the system for 10x/100x/1000x the load.

### **Submission**

Submit the task by creating a private Github repository. Then invite [Dennis](https://github.com/dnnsthnnr) and [Nils](https://github.com/nilaq) to the repository and write us an email, letting us know that you finished. We will review your work as fast as possible and discuss it in a joint session with you.

---

# Additional Resources

**Example of an EDIFACT message**

```xml
UNA:+.? '
UNB+UNOC:3+9904628000007:500+9985046000001:500+251114:2126+866078753PF++VL'
UNH+866078747PF+MSCONS:D:04B:UN:2.4c'
BGM+7+866078747PF+9'
DTM+137:202511142126?+00:303'
RFF+Z13:13017'
NAD+MS+9904628000007::293'
NAD+MR+9985046000001::293'
UNS+D'
NAD+DP'
LOC+172+DE0000801043700000000000011145670'
RFF+MG:31140584'
LIN+1'
PIA+5+1-0?:1.8.0:SRW'
QTY+220:10614'
DTM+9:20251113:102'
DTM+7:202511132300?+00:303'
STS+Z33++Z83'
UNT+17+866078747PF'
UNZ+2+866078753PF'
```

If you want to go deep on the Edifact specifically of the german energy market. The documentation can be found [here](https://bdew-mako.de/documents), be aware of a lot of long German specifications. A Github mirror is also existing: https://github.com/Hochfrequenz/edi_energy_mirror
