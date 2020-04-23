alter table composites_rooms
    drop constraint composites_rooms_pkey;
alter table composites_rooms
    add primary key (composite_id, room_id, gateway_id, position);
