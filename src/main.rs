use rusoto_core::region::Region;
use rusoto_route53::{
    Change, ChangeBatch, ChangeResourceRecordSetsRequest, ResourceRecord, ResourceRecordSet,
    Route53, Route53Client,
};
use std::process::Command;
use structopt::StructOpt;

#[derive(Debug, StructOpt)]
#[structopt(name = "ddns", about = "Simple DDNS")]
struct Opt {
    /// Hosted zone id
    #[structopt(short, long)]
    id: String,

    /// Hosted zone name
    #[structopt(short, long)]
    name: String,
}

struct Entry {
    name: String,
    ip: String,
}

async fn set(zone: String, entries: Vec<Entry>) {
    let region = Region::default();
    let client = Route53Client::new(region);
    let mut changes = vec![];
    for entry in entries {
        println!("Set A record of {} to {}", entry.name, entry.ip);
        changes.push(Change {
            action: "UPSERT".to_string(),
            resource_record_set: ResourceRecordSet {
                ttl: Some(300),
                type_: "A".to_string(),
                name: entry.name,
                resource_records: Some(vec![ResourceRecord { value: entry.ip }]),
                ..ResourceRecordSet::default()
            },
        });
    }
    let resp = client
        .change_resource_record_sets(ChangeResourceRecordSetsRequest {
            hosted_zone_id: zone,
            change_batch: ChangeBatch {
                changes,
                comment: None,
            },
        })
        .await;
    println!("Set dns done with {:?}", resp);
}

#[tokio::main]
async fn main() {
    let opt = Opt::from_args();
    let hostname = String::from_utf8(
        Command::new("hostname")
            .output()
            .expect("run hostname")
            .stdout,
    )
    .expect("hostname should be valid utf8");
    println!("Hostname is {}", hostname.trim());
    let ip = String::from_utf8(
        Command::new("curl")
            .arg("-4")
            .arg("me.gandi.net")
            .output()
            .expect("run curl")
            .stdout,
    )
    .expect("curl should be valid utf8");
    println!("My IP is {}", ip.trim());
    set(
        opt.id,
        vec![Entry {
            name: format!("{}.{}", hostname.trim(), opt.name),
            ip: ip.trim().to_string(),
        }],
    )
    .await;
}
