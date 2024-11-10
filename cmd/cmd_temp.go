package cmd

// const (
// 	address     = "localhost:17078"
// 	defaultName = "world"
// )

// // func init() {
// // 	if len(os.Args) > 1 && os.Args[1] == "test" {
// // 		conn, err := grpc.Dial(address, grpc.WithInsecure())
// // 		if err != nil {
// // 			log.Fatalf("did not connect: %v", err)
// 		// }
// 		// defer conn.Close()
// 		// c := hcore.NewCoreClient(conn)

// 		// ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
// 		// defer cancel()
// 		ctx := context.Background()

// 		// SayHello
// 		stream, err := c.OutboundsInfo(ctx, &hcommon.Empty{})

// 		for {
// 			r, err := stream.Recv()
// 			if err != nil {
// 				log.Fatalf("could not receive: %v", err)
// 			}
// 			log.Printf("Received1: %s", r.String())

// 			time.Sleep(1 * time.Second)
// 		}
// 	}
// }
