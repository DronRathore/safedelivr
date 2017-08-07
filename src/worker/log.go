/*
  package Log
  Handles Log events pushed on the queue.
  Whenever a webhook event is recieved it is analysed for the state
  returned in it, it might happen that few emails are dropped or failed
  in during transmission by a Email service provider in that case we can
  retry sending the same email using the original content of the Batch from
  another provider.

  The only caveat of this functionality is, email could have been dropped because
  of failed MX resolve in which case we will be wasting our bandwidth(todo: resolve MX)
  Emails can also be dropped if the gateway flags your email as a spam, in that case
  we should not aggresively retry sending the message as it will degrade the reputation
  and may cause your Email server to be flagged as a spammer.

*/
package worker
